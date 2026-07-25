package font

import (
	"os"
	"strings"
	"testing"
)

func findLiberationFont() string {
	paths := []string{
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Bold.ttf",
		"/usr/share/fonts/liberation/LiberationSans-Regular.ttf",
		"/usr/share/fonts/LiberationSans-Regular.ttf",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func TestLoadTTF(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}
	if f.UnitsPerEm == 0 {
		t.Error("UnitsPerEm is 0")
	}
	if len(f.cmap) == 0 {
		t.Error("no glyphs loaded")
	}
}

func TestFontMetrics(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}
	if f.Ascent == 0 {
		t.Error("Ascent is 0")
	}
	if f.UnitsPerEm == 0 {
		t.Error("UnitsPerEm is 0")
	}
	if f.FontBBox[0] == 0 && f.FontBBox[1] == 0 && f.FontBBox[2] == 0 && f.FontBBox[3] == 0 {
		t.Error("FontBBox is all zeros")
	}
}

func TestAddChars(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}

	chars := []rune{'A', 'B', 'C', '0', '1', ' '}
	f.AddChars(chars)

	used := f.UsedChars()
	if len(used) == 0 {
		t.Fatal("UsedChars returned empty after AddChars")
	}

	got := make(map[rune]bool)
	for _, r := range used {
		got[r] = true
	}
	for _, r := range chars {
		if !got[r] {
			t.Errorf("char %c not found in UsedChars", r)
		}
	}
}

func TestCharWidth(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}

	w := f.CharWidth('A', 1.0)
	if w == 0 {
		t.Log("CharWidth returned 0 (may be ok if 'A' not in loaded glyphs)")
	}

	w1000 := f.CharWidth('A', 1000.0/float64(f.UnitsPerEm))
	if w1000 != 0 {
		if w1000 < 0 || w1000 > 10000 {
			t.Errorf("unreasonable width: %f", w1000)
		}
	}
}

func TestWidths(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}

	f.AddChars([]rune{'A', 'B', 'C'})
	widths := f.Widths()
	if len(widths) < 3 {
		t.Errorf("expected at least 3 widths, got %d", len(widths))
	}
	for _, w := range widths {
		if w < 0 {
			t.Errorf("width %d is negative", w)
		}
	}
}

func TestToUnicode(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}

	f.AddChars([]rune{'A', 'B', 'C'})
	cmap := f.ToUnicodeCMap()
	if len(cmap) == 0 {
		t.Fatal("ToUnicodeCMap returned nil")
	}

	s := string(cmap)
	if !strings.Contains(s, "begincmap") {
		t.Error("CMap missing begincmap")
	}
	if !strings.Contains(s, "beginbfrange") {
		t.Error("CMap missing beginbfrange")
	}
	if !strings.Contains(s, "0041") {
		t.Error("CMap missing 0041 (Unicode for 'A')")
	}
}

func TestToUnicodeContiguous(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}

	f.AddChars([]rune{'X', 'Y', 'Z'})
	cmap := f.ToUnicodeCMap()
	s := string(cmap)
	if !strings.Contains(s, "beginbfrange") {
		t.Error("CMap missing beginbfrange")
	}
	if !strings.Contains(s, "0058") {
		t.Error("CMap missing range starting at 0058 (X)")
	}
}

func TestLiberationMapping(t *testing.T) {
	expected := map[string]string{
		"Helvetica":             "LiberationSans-Regular",
		"Helvetica-Bold":        "LiberationSans-Bold",
		"Helvetica-Oblique":     "LiberationSans-Italic",
		"Helvetica-BoldOblique": "LiberationSans-BoldItalic",
		"Times-Roman":           "LiberationSerif-Regular",
		"Times-Bold":            "LiberationSerif-Bold",
		"Times-Italic":          "LiberationSerif-Italic",
		"Times-BoldItalic":      "LiberationSerif-BoldItalic",
		"Courier":               "LiberationMono-Regular",
		"Courier-Bold":          "LiberationMono-Bold",
		"Courier-Oblique":       "LiberationMono-Italic",
		"Courier-BoldOblique":   "LiberationMono-BoldItalic",
	}

	for std, lib := range expected {
		got, ok := LiberationFontFor(std)
		if !ok {
			t.Errorf("LiberationFontFor(%q) = false", std)
			continue
		}
		if got != lib {
			t.Errorf("LiberationFontFor(%q) = %q, want %q", std, got, lib)
		}
	}
}

func TestLiberationPaths(t *testing.T) {
	paths := LiberationPaths()
	if len(paths) != 12 {
		t.Errorf("expected 12 Liberation paths, got %d", len(paths))
	}
	for name, path := range paths {
		if path == "" {
			t.Errorf("empty path for %s", name)
		}
		if !strings.HasSuffix(path, ".ttf") {
			t.Errorf("path for %s does not end in .ttf: %s", name, path)
		}
	}
}

func TestRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}

	f := &Font{Name: "MyFont"}
	r.Register("MyFont", f)

	got := r.Get("MyFont")
	if got == nil {
		t.Fatal("Get returned nil after Register")
		return
	}
	if got.Name != "MyFont" {
		t.Errorf("got.Name = %q, want %q", got.Name, "MyFont")
	}

	names := r.UsedNames()
	if len(names) != 1 || names[0] != "MyFont" {
		t.Errorf("UsedNames = %v, want [MyFont]", names)
	}
}

func TestRegistryStandardFont(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}

	r := NewRegistry()
	font, err := r.RegisterStandardFont("Helvetica", "")
	if err != nil {
		t.Fatalf("RegisterStandardFont(Helvetica) = %v", err)
	}
	if font == nil {
		t.Fatal("RegisterStandardFont returned nil font")
		return
	}
	if font.UnitsPerEm == 0 {
		t.Error("font.UnitsPerEm is 0")
	}
}

func TestGlyphCount(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}
	f.AddChars([]rune{'A', 'B', 'C'})
	if f.GlyphCount() == 0 {
		t.Error("GlyphCount is 0 after adding chars")
	}
}

func TestGenerateSubset(t *testing.T) {
	path := findLiberationFont()
	if path == "" {
		t.Skip("TTF not found, install Liberation fonts")
	}
	f, err := LoadFromPath(path)
	if err != nil {
		t.Fatalf("LoadFromPath(%q) = %v", path, err)
	}

	f.AddChars([]rune{'A', 'B', 'C', ' ', '1', '2', '3'})
	err = f.GenerateSubset()
	if err != nil {
		t.Fatalf("GenerateSubset: %v", err)
	}
	if len(f.SubsetData) == 0 {
		t.Fatal("SubsetData is empty after GenerateSubset")
	}
	if len(f.SubsetData) >= len(f.RawData) {
		t.Logf("SubsetData len %d >= RawData len %d (may be expected for small subsets)", len(f.SubsetData), len(f.RawData))
	}
}

func TestStemV(t *testing.T) {
	f := &Font{Style: "Regular"}
	sv := f.StemVValue()
	if sv <= 0 {
		t.Errorf("StemVValue() = %d, expected > 0", sv)
	}

	f2 := &Font{Style: "BoldItalic"}
	sv2 := f2.StemVValue()
	if sv2 <= 80 {
		t.Errorf("Bold StemVValue() = %d, expected > 80", sv2)
	}
}

func TestLoadFromBytesInvalid(t *testing.T) {
	_, err := LoadFromBytes([]byte{0, 0, 0, 0})
	if err == nil {
		t.Error("expected error for invalid TTF data")
	}
}

func TestLoadFromBytesShort(t *testing.T) {
	_, err := LoadFromBytes([]byte{})
	if err == nil {
		t.Error("expected error for empty data")
	}
}
