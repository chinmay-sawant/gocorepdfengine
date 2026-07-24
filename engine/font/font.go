package font

import "sort"

type Font struct {
	Name        string
	Family      string
	Style       string
	IsSerif     bool
	IsMono      bool
	UnitsPerEm  uint16
	Ascent      int16
	Descent     int16
	CapHeight   int16
	FontBBox    [4]int16
	ItalicAngle float64
	StemV       int16
	XHeight     int16
	Flags       uint32
	Glyphs      map[rune]*Glyph
	RawData     []byte
	SubsetData  []byte
	SubGIDMap   map[uint16]uint16

	cmap         map[rune]uint16
	glyphMetrics map[uint16]*Glyph
}

type Glyph struct {
	GID   uint16
	Width int16
	BBox  [4]int16
}

func (f *Font) GlyphCount() int {
	return len(f.Glyphs)
}

func (f *Font) CharWidth(r rune, scale float64) float64 {
	g, ok := f.Glyphs[r]
	if !ok {
		return 0
	}
	if f.UnitsPerEm == 0 {
		return 0
	}
	return float64(g.Width) * scale / float64(f.UnitsPerEm)
}

func (f *Font) Widths() []int {
	if f.UnitsPerEm == 0 {
		return nil
	}

	keys := make([]rune, 0, len(f.Glyphs))
	for r := range f.Glyphs {
		keys = append(keys, r)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	scale := 1000.0 / float64(f.UnitsPerEm)
	widths := make([]int, len(keys))
	for i, r := range keys {
		g := f.Glyphs[r]
		widths[i] = int(float64(g.Width)*scale + 0.5)
	}
	return widths
}

func (f *Font) UsedRunes() []rune {
	keys := make([]rune, 0, len(f.Glyphs))
	for r := range f.Glyphs {
		keys = append(keys, r)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})
	return keys
}
