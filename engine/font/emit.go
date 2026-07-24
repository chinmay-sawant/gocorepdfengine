package font

import (
	"fmt"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

func FontDict(baseFont string, cidFontRef, toUnicodeRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/Type":            "/Font",
		"/Subtype":         "/Type0",
		"/BaseFont":        "/" + baseFont,
		"/Encoding":        "/Identity-H",
		"/DescendantFonts": []interface{}{fmt.Sprintf("%d 0 R", cidFontRef)},
		"/ToUnicode":       fmt.Sprintf("%d 0 R", toUnicodeRef),
	}
}

func CIDFontDict(font *Font, descriptorRef, cidToGIDRef doc.ObjectID) map[string]interface{} {
	d := map[string]interface{}{
		"/Type":           "/Font",
		"/Subtype":        "/CIDFontType2",
		"/BaseFont":       "/" + font.Name,
		"/CIDSystemInfo":  CIDSystemInfoDict("Adobe", "Identity", 0),
		"/FontDescriptor": fmt.Sprintf("%d 0 R", descriptorRef),
		"/DW":             1000,
	}
		if w := WidthsArray(font); w != nil {
		d["/W"] = w
	}
	if cidToGIDRef == 0 {
		d["/CIDToGIDMap"] = "/Identity"
	} else {
		d["/CIDToGIDMap"] = fmt.Sprintf("%d 0 R", cidToGIDRef)
	}
	return d
}

func FontDescriptorDict(font *Font, fontFile2Ref doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/Type":        "/FontDescriptor",
		"/FontName":    "/" + font.Name,
		"/Flags":       int(font.Flags),
		"/FontBBox":    []int{int(font.FontBBox[0]), int(font.FontBBox[1]), int(font.FontBBox[2]), int(font.FontBBox[3])},
		"/ItalicAngle": font.ItalicAngle,
		"/Ascent":      int(font.Ascent),
		"/Descent":     int(font.Descent),
		"/CapHeight":   int(font.CapHeight),
		"/StemV":       int(font.StemVValue()),
		"/XHeight":     int(font.XHeight),
		"/FontFile2":   fmt.Sprintf("%d 0 R", fontFile2Ref),
	}
}

func CIDSystemInfoDict(registry, ordering string, supplement int) map[string]interface{} {
	return map[string]interface{}{
		"/Registry":   fmt.Sprintf("(%s)", registry),
		"/Ordering":   fmt.Sprintf("(%s)", ordering),
		"/Supplement": supplement,
	}
}

func WidthsArray(font *Font) []int {
	keys := font.UsedRunes()
	if len(keys) == 0 {
		return nil
	}
	scale := 1000.0 / float64(font.UnitsPerEm)

	// Groups of contiguous CIDs that share the same width
	type cidRange struct{ first, last, width int }
	var ranges []cidRange

	for _, r := range keys {
		cid := int(r)
		g, ok := font.Glyphs[r]
		if !ok {
			continue
		}
		width := int(float64(g.Width)*scale + 0.5)
		if n := len(ranges); n > 0 && ranges[n-1].width == width && ranges[n-1].last+1 == cid {
			ranges[n-1].last = cid
		} else {
			ranges = append(ranges, cidRange{cid, cid, width})
		}
	}

	result := make([]int, 0, len(ranges)*3)
	for _, r := range ranges {
		result = append(result, r.first, r.last, r.width)
	}
	return result
}
