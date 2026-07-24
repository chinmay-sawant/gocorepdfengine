package engine

import (
	"github.com/chinmay/gocorepdfengine/engine/color"
	"github.com/chinmay/gocorepdfengine/engine/content"
	"github.com/chinmay/gocorepdfengine/engine/doc"
	"github.com/chinmay/gocorepdfengine/engine/font"
	"github.com/chinmay/gocorepdfengine/engine/meta"
	"github.com/chinmay/gocorepdfengine/engine/page"
	"github.com/chinmay/gocorepdfengine/engine/pdfa"
	"github.com/chinmay/gocorepdfengine/engine/structure"
	"github.com/chinmay/gocorepdfengine/engine/write"
)

// PageContent is one page stream plus font resource labels used on that page.
type PageContent struct {
	Stream    []byte
	FontRes   map[string]string // logical font name -> /F1 label
	UsedFonts map[string]bool
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
}

// GenerateDocument assembles a multi-page PDF 2.0 document, optionally PDF/A-4 + PDF/UA-2.
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
	contentIDs := make([]doc.ObjectID, len(cfg.Pages))
	for i := range contentIDs {
		contentIDs[i] = d.AllocID()
	}

	// One Type1 or Type0 font object per logical name (map label F1.. shared).
	// We emit a single Helvetica/Liberation resource as F1 for simplicity and
	// rewrite is not needed if layout already used F1 for first font.
	type fontChain struct {
		fontRef, cidFontID, descriptorID, fontFile2ID, toUnicodeID, cidToGIDMapID doc.ObjectID
		resourceLabel                                                              string
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
	for i, pc := range cfg.Pages {
		streamBytes := pc.Stream
		// Ensure layout font labels resolve: if stream uses /F1 we map F1 → shared font.
		d.AddObjectAt(contentIDs[i], &write.Stream{
			Dict: map[string]interface{}{"/Length": len(streamBytes)},
			Data: streamBytes,
		})
	}

	// === Font ===
	if isA4 {
		reg := font.NewRegistry()
		loadedFont, err := reg.RegisterStandardFont("Helvetica", "")
		if err != nil {
			loadedFont, err = font.LoadFromPath("/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf")
			if err != nil {
				loadedFont = nil
			}
		}
		if loadedFont != nil {
			for _, r := range cfg.UsedText {
				loadedFont.AddChar(r)
			}
			// Also mark digits/common punctuation.
			for _, r := range "0123456789.,:/-₹ " {
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
			d.AddObjectAt(shared.descriptorID, font.FontDescriptorDict(loadedFont, shared.fontFile2ID))
			d.AddObjectAt(shared.cidFontID, font.CIDFontDict(loadedFont, shared.descriptorID, shared.cidToGIDMapID))
			d.AddObjectAt(shared.fontRef, font.FontDict(libName, shared.cidFontID, shared.toUnicodeID))
		} else {
			// Minimal fallback chain
			fake := &font.Font{
				Name: "LiberationSans-Regular", Flags: 32,
				FontBBox: [4]int16{-1000, -1000, 1000, 1000},
				Ascent: 1000, Descent: -200, CapHeight: 700, StemV: 80, XHeight: 500,
			}
			d.AddObjectAt(shared.fontFile2ID, &write.Stream{Dict: map[string]interface{}{"/Length": 0}, Data: []byte{}})
			tuData := []byte("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\nendcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
			d.AddObjectAt(shared.toUnicodeID, &write.Stream{Dict: map[string]interface{}{"/Length": len(tuData)}, Data: tuData})
			d.AddObjectAt(shared.descriptorID, font.FontDescriptorDict(fake, shared.fontFile2ID))
			d.AddObjectAt(shared.cidFontID, font.CIDFontDict(fake, shared.descriptorID, 0))
			d.AddObjectAt(shared.fontRef, font.FontDict(fake.Name, shared.cidFontID, shared.toUnicodeID))
		}
	} else {
		d.AddObjectAt(shared.fontRef, map[string]interface{}{
			"/Type": "/Font", "/Subtype": "/Type1", "/BaseFont": "/Helvetica",
		})
	}

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
		d.AddObjectAt(pageIDs[i], pg.ToDict(pg.FontResources, nil, pagesID))
	}

	p := page.NewPages()
	p.Kids = pageIDs
	p.Count = len(pageIDs)
	d.AddObjectAt(pagesID, p.ToDict())
	d.SetPagesRoot(pagesID)

	// === A-4 / UA metadata ===
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
	} else if isUA {
		xmpCfg := meta.DefaultConfig()
		xmpCfg.PDFUA = true
		xmpCfg.Title = cfg.Title
		xmpCfg.Author = cfg.Author
		xmpCfg.Subject = cfg.Subject
		xmpCfg.Creator = cfg.Creator
		metaDict, metaData := meta.MetadataStream(xmpCfg)
		d.AddObjectAt(metaRef, &write.Stream{Dict: metaDict, Data: metaData})
	}

	// === Structure tree (simple: Document → one P per page) ===
	if isUA {
		d.AddObjectAt(nsRef, structure.Namespace())
		kids := make([]structure.StructElemKid, 0, len(cfg.Pages))
		ptMap := make(map[int][]doc.ObjectID, len(cfg.Pages))
		for i := range cfg.Pages {
			pElem := &structure.StructElem{
				Type:    structure.S_P,
				Parent:  elemDocID,
				PageRef: pageIDs[i],
				MCID:    0,
			}
			d.AddObjectAt(elemPageIDs[i], structure.StructElemDict(pElem))
			kids = append(kids, structure.StructElemKid{Ref: elemPageIDs[i]})
			ptMap[i] = []doc.ObjectID{elemPageIDs[i]}
		}
		docElem := &structure.StructElem{
			Type:         structure.S_Document,
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

	// === Catalog ===
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

	return d.Build(), nil
}
