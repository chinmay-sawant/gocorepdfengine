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
		"/CIDSystemInfo":  CIDSystemInfoDict("Adobe", "Identity", 0),
		"/FontDescriptor": fmt.Sprintf("%d 0 R", descriptorRef),
		"/DW":             1000,
	}
	if w := WidthsArray(font); len(w) > 0 {
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
	widths := font.Widths()
	if len(widths) == 0 {
		return nil
	}
	result := make([]int, 0, len(widths)+2)
	result = append(result, 1, len(widths))
	result = append(result, widths...)
	return result
}
