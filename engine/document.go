// Package engine provides PDF document generation with support for PDF 2.0,
// PDF/A-4, and PDF/UA-2.
package engine

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/color"
	"github.com/chinmay/gocorepdfengine/engine/content"
	"github.com/chinmay/gocorepdfengine/engine/doc"
	"github.com/chinmay/gocorepdfengine/engine/font"
	"github.com/chinmay/gocorepdfengine/engine/image"
	"github.com/chinmay/gocorepdfengine/engine/meta"
	"github.com/chinmay/gocorepdfengine/engine/page"
	"github.com/chinmay/gocorepdfengine/engine/pdfa"
	"github.com/chinmay/gocorepdfengine/engine/structure"
	"github.com/chinmay/gocorepdfengine/engine/write"
)

// PageContent is one page stream plus font/image resource labels used on that page.
type PageContent struct {
	Stream         []byte
	FontRes        map[string]string // logical font name -> /F1 label
	UsedFonts      map[string]bool
	ImageXObjects  map[string]*image.Image // XObject name -> image data
}

type fontChain struct {
	fontRef, cidFontID, descriptorID, fontFile2ID, toUnicodeID, cidToGIDMapID doc.ObjectID
	resourceLabel                                                             string
}

// DocumentConfig drives multi-page generation from pre-built content streams
// (typically produced by package layout / render).
type DocumentConfig struct {
	Width, Height float64
	Mode          doc.Mode
	Title         string
	Author        string
	Subject       string
	Creator       string
	Lang          string
	Pages         []PageContent
	UsedText      string // characters for PDF/A subsetting
	FooterText    string
}

// GenerateDocument builds a complete PDF binary from pre-built page content
// streams, handling font embedding, ICC profiles, structure trees (PDF/UA-2),
// output intents (PDF/A-4), page numbering, and footer text.
func GenerateDocument(cfg DocumentConfig) ([]byte, error) {
	if cfg.Width == 0 {
		cfg.Width = 595
	}
	if cfg.Height == 0 {
		cfg.Height = 842
	}
	if len(cfg.Pages) == 0 {
		// Empty page
		cfg.Pages = []PageContent{{Stream: content.NewStream().Bytes()}}
	}
	if cfg.Creator == "" {
		cfg.Creator = "gocorepdfengine"
	}

	d := doc.NewDocument()
	d.Mode = cfg.Mode
	if d.Mode == 0 {
		d.Mode = doc.ModePDF20
	}

	isA4 := d.HasMode(doc.ModePDFA4)
	isUA := d.HasMode(doc.ModePDFUA2)
	if isA4 {
		d.Mode |= doc.ModeEmbedFonts
		d.TrailerInfo = nil
	}

	lang := cfg.Lang
	if lang == "" && isUA {
		lang = "en-US"
	}

	// Collect font names used across pages.
	fontNames := map[string]bool{}
	for _, p := range cfg.Pages {
		for name := range p.UsedFonts {
			fontNames[name] = true
		}
		for name := range p.FontRes {
			fontNames[name] = true
		}
	}
	if len(fontNames) == 0 {
		fontNames["Helvetica"] = true
	}

	// === Allocate IDs ===
	contentIDs := make([]doc.ObjectID, len(cfg.Pages)) // dense slice, not map — fine
	for i := range contentIDs {
		contentIDs[i] = d.AllocID()
	}

	// Single shared font for v1 multi-page path (layout uses Helvetica → F1).
	var shared fontChain
	shared.resourceLabel = "F1"
	if isA4 {
		shared.fontRef = d.AllocID()
		shared.cidFontID = d.AllocID()
		shared.descriptorID = d.AllocID()
		shared.fontFile2ID = d.AllocID()
		shared.toUnicodeID = d.AllocID()
		shared.cidToGIDMapID = d.AllocID()
	} else {
		shared.fontRef = d.AllocID()
	}

	pageIDs := make([]doc.ObjectID, len(cfg.Pages))
	for i := range pageIDs {
		pageIDs[i] = d.AllocID()
	}
	pagesID := d.AllocID()

	var metaRef, srgbRef, grayRef, oiRef doc.ObjectID
	if isA4 {
		metaRef = d.AllocID()
		srgbRef = d.AllocID()
		grayRef = d.AllocID()
		oiRef = d.AllocID()
	} else if isUA {
		metaRef = d.AllocID()
	}

	var nsRef, strRootRef, ptRef, elemDocID doc.ObjectID
	elemPageIDs := make([]doc.ObjectID, 0, len(cfg.Pages))
	if isUA {
		nsRef = d.AllocID()
		strRootRef = d.AllocID()
		ptRef = d.AllocID()
		elemDocID = d.AllocID()
		for range cfg.Pages {
			elemPageIDs = append(elemPageIDs, d.AllocID())
		}
	}

	catalogID := d.AllocID()

	// === Content streams ===
	buildContentStreams(d, cfg, isUA, contentIDs, strconv.Itoa(len(cfg.Pages)))

	// === Font (cold path, one-time setup) ===
	setupDocumentFont(d, cfg.UsedText, isA4, &shared)

	// === Pages ===
	for i := range cfg.Pages {
		pg := page.NewPage(cfg.Width, cfg.Height)
		pg.ContentsRef = contentIDs[i]
		// Map every layout label (F1, F2, …) to the shared font for v1.
		fontMap := map[string]doc.ObjectID{}
		if pc := cfg.Pages[i]; len(pc.FontRes) > 0 {
			for _, label := range pc.FontRes {
				fontMap[label] = shared.fontRef
			}
		} else {
			fontMap["F1"] = shared.fontRef
		}
		pg.FontResources = fontMap

		// Image XObjects
		xobjMap := map[string]doc.ObjectID{}
		if pc := cfg.Pages[i]; len(pc.ImageXObjects) > 0 {
			for xObjName, img := range pc.ImageXObjects {
				imgRef := d.AllocID()
				xobjMap[xObjName] = imgRef
				dict := map[string]interface{}{
					"/Type":             "/XObject",
					"/Subtype":          "/Image",
					"/Width":            img.Width,
					"/Height":           img.Height,
					"/ColorSpace":       img.ColorSpace,
					"/BitsPerComponent": img.BitsPerComponent,
					"/Filter":           img.Filter,
				}
				d.AddObjectAt(imgRef, &write.Stream{Dict: dict, Data: img.Data})
			}
		}
		pg.XObjectResources = xobjMap

		if isA4 {
			pg.ColorSpaceResources = map[string]interface{}{
				"/DefaultRGB":  []interface{}{"/ICCBased", write.Ref(int(srgbRef), 0)},
				"/DefaultGray": []interface{}{"/ICCBased", write.Ref(int(grayRef), 0)},
			}
		}
		if isUA {
			sp := structure.StructParentsValue(i)
			pg.StructParents = &sp
		}
		d.AddObjectAt(pageIDs[i], pg.ToDict(fontMap, xobjMap, pagesID))
	}

	p := page.NewPages()
	p.Kids = pageIDs
	p.Count = len(pageIDs)
	d.AddObjectAt(pagesID, p.ToDict())
	d.SetPagesRoot(pagesID)

	// === A-4 / UA metadata ===
	addComplianceMetadata(d, cfg, isA4, isUA, metaRef, srgbRef, grayRef, oiRef)

	// === Structure tree (simple: Document → one P per page) ===
	buildStructureTree(d, isUA, cfg.Pages, lang, nsRef, strRootRef, ptRef, elemDocID, elemPageIDs, pageIDs)

	// === Catalog ===
	buildCatalog(d, catalogID, pagesID, metaRef, strRootRef, oiRef, lang, isA4, isUA)

	return d.Build(), nil
}

func buildContentStreams(d *doc.Document, cfg DocumentConfig, isUA bool, contentIDs []doc.ObjectID, totalPagesStr string) {
	totalPages := len(cfg.Pages)
	footerX := strconv.FormatFloat(cfg.Width*0.02, 'f', -1, 64)
	footerY := strconv.FormatFloat(cfg.Height*0.02, 'f', -1, 64)
	var pageBuf []byte
	for i, pc := range cfg.Pages {
		streamBytes := pc.Stream
		if isUA {
			streamBytes = append([]byte("/P <</MCID 0>> BDC\n"), streamBytes...)
		}
		if cfg.FooterText != "" || totalPages > 1 {
			var buf bytes.Buffer
			buf.Grow(256)
			pageNum := i + 1
			if isUA {
				buf.WriteString("/Artifact BMC\n")
			}

			if cfg.FooterText != "" {
				buf.WriteString("BT /F1 8 Tf 0.5 0.5 0.5 rg ")
				buf.WriteString(footerX)
				buf.WriteString(" ")
				buf.WriteString(footerY)
				buf.WriteString(" Td <")
				for _, r := range cfg.FooterText {
					fmt.Fprintf(&buf, "%04X", r)
				}
				buf.WriteString("> Tj ET\n")
			}

			pageBuf = strconv.AppendInt(pageBuf[:0], int64(pageNum), 10)
			pageStr := "Page " + string(pageBuf) + " of " + totalPagesStr
			buf.WriteString("BT /F1 8 Tf 0.5 0.5 0.5 rg ")
			pageW := float64(len(pageStr)) * 8 * 0.55
			pageBuf = strconv.AppendFloat(pageBuf[:0], cfg.Width*0.98-pageW, 'f', 6, 64)
			buf.Write(pageBuf)
			buf.WriteString(" ")
			buf.WriteString(footerY)
			buf.WriteString(" Td <")
			for _, r := range pageStr {
				fmt.Fprintf(&buf, "%04X", r)
			}
			buf.WriteString("> Tj ET\n")
			if isUA {
				buf.WriteString("EMC\n")
			}

			streamBytes = append(append([]byte{}, streamBytes...), buf.Bytes()...)
		}
		if isUA {
			streamBytes = append(streamBytes, []byte("EMC\n")...)
		}
		d.AddObjectAt(contentIDs[i], &write.Stream{
			Dict: map[string]interface{}{"/Length": len(streamBytes)},
			Data: streamBytes,
		})
	}
}

func setupDocumentFont(d *doc.Document, usedText string, isA4 bool, shared *fontChain) {
	if !isA4 {
		d.AddObjectAt(shared.fontRef, map[string]interface{}{
			"/Type": "/Font", "/Subtype": "/Type1", "/BaseFont": "/Helvetica",
		})
		return
	}

	reg := font.NewRegistry()
	loadedFont, err := reg.RegisterStandardFont("Helvetica", "")
	if err != nil {
		loadedFont, err = font.LoadFromPath("/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf")
		if err != nil {
			loadedFont = nil
		}
	}
	if loadedFont != nil {
		for _, r := range usedText {
			loadedFont.AddChar(r)
		}
		for _, r := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789.,:/-₹ |()$#%&*+<=>?@[]{!}_" {
			loadedFont.AddChar(r)
		}
		libName := loadedFont.Name
		compressed := compressData(loadedFont.RawData)
		d.AddObjectAt(shared.fontFile2ID, &write.Stream{
			Dict: map[string]interface{}{"/Length": len(compressed), "/Filter": "/FlateDecode"},
			Data: compressed,
		})
		tuData := loadedFont.ToUnicodeCMap()
		d.AddObjectAt(shared.toUnicodeID, &write.Stream{
			Dict: map[string]interface{}{"/Length": len(tuData)},
			Data: tuData,
		})
		cidMapData := loadedFont.BuildCIDToGIDMap()
		compressedMap := compressData(cidMapData)
		d.AddObjectAt(shared.cidToGIDMapID, &write.Stream{
			Dict: map[string]interface{}{"/Length": len(compressedMap), "/Filter": "/FlateDecode"},
			Data: compressedMap,
		})
		d.AddObjectAt(shared.descriptorID, font.DescriptorDict(loadedFont, shared.fontFile2ID))
		d.AddObjectAt(shared.cidFontID, font.CIDFontDict(loadedFont, shared.descriptorID, shared.cidToGIDMapID))
		d.AddObjectAt(shared.fontRef, font.Dict(libName, shared.cidFontID, shared.toUnicodeID))
		return
	}

	fake := &font.Font{
		Name: "LiberationSans-Regular", Flags: 32,
		FontBBox: [4]int16{-1000, -1000, 1000, 1000},
		Ascent: 1000, Descent: -200, CapHeight: 700, StemV: 80, XHeight: 500,
	}
	d.AddObjectAt(shared.fontFile2ID, &write.Stream{Dict: map[string]interface{}{"/Length": 0}, Data: []byte{}})
	tuData := []byte("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\nendcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
	d.AddObjectAt(shared.toUnicodeID, &write.Stream{Dict: map[string]interface{}{"/Length": len(tuData)}, Data: tuData})
	d.AddObjectAt(shared.descriptorID, font.DescriptorDict(fake, shared.fontFile2ID))
	d.AddObjectAt(shared.cidFontID, font.CIDFontDict(fake, shared.descriptorID, 0))
	d.AddObjectAt(shared.fontRef, font.Dict(fake.Name, shared.cidFontID, shared.toUnicodeID))
}

func buildStructureTree(d *doc.Document, isUA bool, pages []PageContent, lang string, nsRef, strRootRef, ptRef, elemDocID doc.ObjectID, elemPageIDs, pageIDs []doc.ObjectID) {
	if !isUA {
		return
	}
	d.AddObjectAt(nsRef, structure.Namespace())
	kids := make([]structure.StructElemKid, 0, len(pages))
	ptMap := make(map[int][]doc.ObjectID, len(pages))
	for i := range pages {
		pElem := &structure.StructElem{
			Type:    structure.TypeP,
			Parent:  elemDocID,
			PageRef: pageIDs[i],
			MCID:    0,
		}
		d.AddObjectAt(elemPageIDs[i], structure.StructElemDict(pElem))
		kids = append(kids, structure.StructElemKid{Ref: elemPageIDs[i]})
		ptMap[i] = []doc.ObjectID{elemPageIDs[i]}
	}
	docElem := &structure.StructElem{
		Type:         structure.TypeDocument,
		ObjectID:     elemDocID,
		Parent:       strRootRef,
		NamespaceRef: nsRef,
		Lang:         lang,
		MCID:         -1,
		Kids:         kids,
	}
	d.AddObjectAt(elemDocID, structure.StructElemDict(docElem))
	d.AddObjectAt(ptRef, structure.ParentTreeDict(ptMap, nil))
	d.AddObjectAt(strRootRef, structure.StructTreeRootDict(elemDocID, ptRef, nsRef))
}

func addComplianceMetadata(d *doc.Document, cfg DocumentConfig, isA4, isUA bool, metaRef, srgbRef, grayRef, oiRef doc.ObjectID) {
	if isA4 {
		d.AddObjectAt(srgbRef, &write.Stream{Dict: color.SRGBProfileDict(), Data: color.SRGBProfile()})
		d.AddObjectAt(grayRef, &write.Stream{Dict: color.GrayProfileDict(), Data: color.GrayProfile()})
		d.AddObjectAt(oiRef, pdfa.OutputIntentDict(srgbRef))
		xmpCfg := meta.DefaultConfig()
		xmpCfg.PDFA = true
		xmpCfg.PDFUA = isUA
		xmpCfg.Title = cfg.Title
		xmpCfg.Author = cfg.Author
		xmpCfg.Subject = cfg.Subject
		xmpCfg.Creator = cfg.Creator
		metaDict, metaData := meta.MetadataStream(xmpCfg)
		d.AddObjectAt(metaRef, &write.Stream{Dict: metaDict, Data: metaData})
		return
	}
	if isUA {
		xmpCfg := meta.DefaultConfig()
		xmpCfg.PDFUA = true
		xmpCfg.Title = cfg.Title
		xmpCfg.Author = cfg.Author
		xmpCfg.Subject = cfg.Subject
		xmpCfg.Creator = cfg.Creator
		metaDict, metaData := meta.MetadataStream(xmpCfg)
		d.AddObjectAt(metaRef, &write.Stream{Dict: metaDict, Data: metaData})
	}
}

func buildCatalog(d *doc.Document, catalogID, pagesID, metaRef, strRootRef, oiRef doc.ObjectID, lang string, isA4, isUA bool) {
	catalogDict := map[string]interface{}{
		"/Type":  "/Catalog",
		"/Pages": write.Ref(int(pagesID), 0),
	}
	if lang != "" {
		catalogDict["/Lang"] = "(" + lang + ")"
	}
	if isA4 || isUA {
		catalogDict["/Metadata"] = write.Ref(int(metaRef), 0)
	}
	if isUA {
		catalogDict["/MarkInfo"] = map[string]interface{}{"/Marked": true}
		catalogDict["/ViewerPreferences"] = map[string]interface{}{"/DisplayDocTitle": true}
		catalogDict["/StructTreeRoot"] = write.Ref(int(strRootRef), 0)
	}
	if isA4 {
		catalogDict["/OutputIntents"] = []interface{}{write.Ref(int(oiRef), 0)}
	}
	d.AddObjectAt(catalogID, catalogDict)
	d.SetCatalog(catalogID)
}
