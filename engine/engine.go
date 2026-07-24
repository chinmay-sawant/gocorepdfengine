package engine

import (
	"bytes"
	"compress/zlib"

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

func compressData(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(data)
	w.Close()
	return buf.Bytes()
}

type Config struct {
	Width    float64
	Height   float64
	Font     string
	FontSize float64
	Text     string
	Mode     doc.Mode
	Lang     string
	Title    string
	Author   string
	Subject  string
	Creator  string
}

type Result struct {
	Data []byte
}

func Generate(config Config) (Result, error) {
	d := doc.NewDocument()
	if config.Mode != 0 {
		d.Mode = config.Mode
	}

	isA4 := d.HasMode(doc.ModePDFA4)
	isUA := d.HasMode(doc.ModePDFUA2)
	if isA4 {
		d.Mode |= doc.ModeEmbedFonts
		d.TrailerInfo = nil
	}

	lang := config.Lang
	if lang == "" && isUA {
		lang = "en-US"
	}
	if config.Creator == "" {
		config.Creator = "gocorepdfengine"
	}

	// === Allocate all object IDs first ===
	contentStreamID := d.AllocID()

	var fontRef, cidFontID, descriptorID, fontFile2ID, toUnicodeID, cidToGIDMapID doc.ObjectID
	if isA4 {
		fontRef = d.AllocID()
		cidFontID = d.AllocID()
		descriptorID = d.AllocID()
		fontFile2ID = d.AllocID()
		toUnicodeID = d.AllocID()
		cidToGIDMapID = d.AllocID()
	} else {
		fontRef = d.AllocID()
	}

	pageID := d.AllocID()
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
	if isUA {
		nsRef = d.AllocID()
		strRootRef = d.AllocID()
		ptRef = d.AllocID()
		elemDocID = d.AllocID()
	}

	catalogID := d.AllocID()

	// === Content stream ===
	s := content.NewStream()
	if isUA {
		s.BDC("P", 0)
	}
	s.BT()
	s.Tf("F1", config.FontSize)
	s.Td(72, config.Height-150)
	if isA4 {
		s.TjCID(config.Text)
	} else {
		s.Tj(config.Text)
	}
	s.ET()
	if isUA {
		s.EMC()
	}
	streamBytes := s.Bytes()

	d.AddObjectAt(contentStreamID, &write.Stream{
		Dict: map[string]interface{}{"/Length": len(streamBytes)},
		Data: streamBytes,
	})

	// === Font objects ===
	fontName := config.Font
	if fontName == "" {
		fontName = "Helvetica"
	}

	if isA4 {
		reg := font.NewRegistry()
		loadedFont, err := reg.RegisterStandardFont(fontName, "")
		if err != nil {
			// Fallback: use LiberationSans-Regular directly
			loadedFont, err = font.LoadFromPath("/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf")
			if err != nil {
				// Last resort: use a minimal hardcoded font name
				loadedFont = nil
			}
		}

		if loadedFont != nil {
			libName := loadedFont.Name

			// Mark all used characters from the content
			for _, r := range config.Text {
				loadedFont.AddChar(r)
			}

			fontFile2Data := loadedFont.RawData

			// FontFile2 stream with compressed font data
			compressed := compressData(fontFile2Data)
			d.AddObjectAt(fontFile2ID, &write.Stream{
				Dict: map[string]interface{}{
					"/Length": len(compressed),
					"/Filter": "/FlateDecode",
				},
				Data: compressed,
			})

			// ToUnicode CMap
			tuData := loadedFont.ToUnicodeCMap()
			d.AddObjectAt(toUnicodeID, &write.Stream{
				Dict: map[string]interface{}{"/Length": len(tuData)},
				Data: tuData,
			})

			// CIDToGIDMap stream mapping CIDs → subset GIDs
			cidMapData := loadedFont.BuildCIDToGIDMap()
			compressedMap := compressData(cidMapData)
			d.AddObjectAt(cidToGIDMapID, &write.Stream{
				Dict: map[string]interface{}{
					"/Length": len(compressedMap),
					"/Filter": "/FlateDecode",
				},
				Data: compressedMap,
			})

			// Use font emit functions
			d.AddObjectAt(descriptorID, font.FontDescriptorDict(loadedFont, fontFile2ID))
			d.AddObjectAt(cidFontID, font.CIDFontDict(loadedFont, descriptorID, cidToGIDMapID))
			d.AddObjectAt(fontRef, font.FontDict(libName, cidFontID, toUnicodeID))
		} else {
			// Fallback: hardcoded minimal font chain
			libName := "LiberationSans-Regular"

			fakeFont := &font.Font{
				Name:       libName,
				Flags:      32,
				FontBBox:   [4]int16{-1000, -1000, 1000, 1000},
				ItalicAngle: 0,
				Ascent:     1000,
				Descent:    -200,
				CapHeight:  700,
				StemV:      80,
				XHeight:    500,
			}

			d.AddObjectAt(fontFile2ID, &write.Stream{
				Dict: map[string]interface{}{"/Length": 0},
				Data: []byte{},
			})
			tuData := []byte("/CIDInit /ProcSet findresource begin\n12 dict begin\nbegincmap\n/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n/CMapName /Adobe-Identity-UCS def\n/CMapType 2 def\n1 begincodespacerange\n<0000> <FFFF>\nendcodespacerange\nendcmap\nCMapName currentdict /CMap defineresource pop\nend\nend\n")
			d.AddObjectAt(toUnicodeID, &write.Stream{Dict: map[string]interface{}{"/Length": len(tuData)}, Data: tuData})
			d.AddObjectAt(descriptorID, font.FontDescriptorDict(fakeFont, fontFile2ID))
			d.AddObjectAt(cidFontID, font.CIDFontDict(fakeFont, descriptorID, 0))
			d.AddObjectAt(fontRef, font.FontDict(libName, cidFontID, toUnicodeID))
		}
	} else {
		d.AddObjectAt(fontRef, map[string]interface{}{
			"/Type":    "/Font",
			"/Subtype": "/Type1",
			"/BaseFont": "/" + fontName,
		})
	}

	// === Page objects ===
	pg := page.NewPage(config.Width, config.Height)
	pg.ContentsRef = contentStreamID
	pg.FontResources = map[string]doc.ObjectID{"F1": fontRef}
	if isA4 {
		pg.ColorSpaceResources = map[string]interface{}{
			"/DefaultRGB":  []interface{}{"/ICCBased", write.Ref(int(srgbRef), 0)},
			"/DefaultGray": []interface{}{"/ICCBased", write.Ref(int(grayRef), 0)},
		}
	}
	if isUA {
		sp := structure.StructParentsValue(0)
		pg.StructParents = &sp
		pg.Tabs = "S"
	}
	d.AddObjectAt(pageID, pg.ToDict(pg.FontResources, nil, pagesID))

	p := page.NewPages()
	p.Kids = []doc.ObjectID{pageID}
	p.Count = 1
	d.AddObjectAt(pagesID, p.ToDict())
	d.SetPagesRoot(pagesID)

	// === A-4 objects (ICC profiles, OutputIntent, XMP metadata) ===
	if isA4 {
		d.AddObjectAt(srgbRef, &write.Stream{
			Dict: color.SRGBProfileDict(),
			Data: color.SRGBProfile(),
		})
		d.AddObjectAt(grayRef, &write.Stream{
			Dict: color.GrayProfileDict(),
			Data: color.GrayProfile(),
		})
		d.AddObjectAt(oiRef, pdfa.OutputIntentDict(srgbRef))

		xmpCfg := meta.DefaultConfig()
		xmpCfg.PDFA = true
		xmpCfg.PDFUA = isUA
		xmpCfg.Title = config.Title
		xmpCfg.Author = config.Author
		xmpCfg.Subject = config.Subject
		xmpCfg.Creator = config.Creator
		metaDict, metaData := meta.MetadataStream(xmpCfg)
		d.AddObjectAt(metaRef, &write.Stream{Dict: metaDict, Data: metaData})
	} else if isUA {
		xmpCfg := meta.DefaultConfig()
		xmpCfg.PDFUA = true
		xmpCfg.Title = config.Title
		xmpCfg.Author = config.Author
		xmpCfg.Subject = config.Subject
		xmpCfg.Creator = config.Creator
		metaDict, metaData := meta.MetadataStream(xmpCfg)
		d.AddObjectAt(metaRef, &write.Stream{Dict: metaDict, Data: metaData})
	}

	// === Structure tree (UA-2) ===
	if isUA {
		nsDict := structure.Namespace()
		d.AddObjectAt(nsRef, nsDict)

		docElem := &structure.StructElem{
			Type:         structure.S_Document,
			ObjectID:     elemDocID,
			Parent:       strRootRef,
			NamespaceRef: nsRef,
			MCID:         -1,
		}
		d.AddObjectAt(elemDocID, structure.StructElemDict(docElem))

		pElem := &structure.StructElem{
			Type:     structure.S_P,
			Parent:   elemDocID,
			PageRef:  pageID,
			MCID:     0,
		}
		pElemID := d.AllocID()
		d.AddObjectAt(pElemID, structure.StructElemDict(pElem))

		docElem.Kids = []structure.StructElemKid{
			{Ref: pElemID},
		}

		parentTreeDict := structure.ParentTreeDict(map[int][]doc.ObjectID{0: {pElemID}}, nil)
		d.AddObjectAt(ptRef, parentTreeDict)

		strRootDict := structure.StructTreeRootDict(elemDocID, ptRef, nsRef)
		d.AddObjectAt(strRootRef, strRootDict)
	}

	// === Catalog (created last with all references) ===
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

	return Result{Data: d.Build()}, nil
}
