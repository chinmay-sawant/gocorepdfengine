package font

import (
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

// Dict returns a raw PDF font dictionary (Type0 / CIDFont) for serialization.
func Dict(baseFont string, cidFontRef, toUnicodeRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/Type":            "/Font",
		"/Subtype":         "/Type0",
		"/BaseFont":        "/" + baseFont,
		"/Encoding":        "/Identity-H",
		"/DescendantFonts": []interface{}{strconv.Itoa(int(cidFontRef)) + " 0 R"},
		"/ToUnicode":       strconv.Itoa(int(toUnicodeRef)) + " 0 R",
	}
}

// CIDFontDict returns a raw PDF CIDFont dictionary for serialization.
func CIDFontDict(font *Font, descriptorRef, cidToGIDRef doc.ObjectID) map[string]interface{} {
	dw := font.DefaultWidth()
	d := map[string]interface{}{
		"/Type":           "/Font",
		"/Subtype":        "/CIDFontType2",
		"/BaseFont":       "/" + font.Name,
		"/CIDSystemInfo":  CIDSystemInfoDict("Adobe", "Identity", 0),
		"/FontDescriptor": strconv.Itoa(int(descriptorRef)) + " 0 R",
		"/DW":             dw,
	}
	if w := WidthsArray(font); w != nil {
		d["/W"] = w
	}
	if cidToGIDRef == 0 {
		d["/CIDToGIDMap"] = "/Identity"
	} else {
		d["/CIDToGIDMap"] = strconv.Itoa(int(cidToGIDRef)) + " 0 R"
	}
	return d
}

// DescriptorDict returns a raw PDF FontDescriptor dictionary for serialization.
func DescriptorDict(font *Font, fontFile2Ref doc.ObjectID) map[string]interface{} {
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
		"/FontFile2":   strconv.Itoa(int(fontFile2Ref)) + " 0 R",
	}
}

// CIDSystemInfoDict returns a raw PDF CIDSystemInfo dictionary for serialization.
func CIDSystemInfoDict(registry, ordering string, supplement int) map[string]interface{} {
	return map[string]interface{}{
		"/Registry":   "(" + registry + ")",
		"/Ordering":   "(" + ordering + ")",
		"/Supplement": supplement,
	}
}

// WidthsArray builds the /W array for a CIDFont, grouping contiguous CIDs with the same width.
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
