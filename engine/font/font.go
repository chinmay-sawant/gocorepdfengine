package font

import (
	"encoding/binary"
	"sort"
)

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

// GlyphCount returns the number of glyphs in the font.
func (f *Font) GlyphCount() int {
	return len(f.Glyphs)
}

// CharWidth returns the width of the given rune at the given scale factor.
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

// Widths returns the widths of all glyphs, scaled to a 1000-unit EM.
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

	scale := ttfUPEm / float64(f.UnitsPerEm)
	widths := make([]int, len(keys))
	for i, r := range keys {
		g := f.Glyphs[r]
		widths[i] = int(float64(g.Width)*scale + roundingHalf)
	}
	return widths
}

// DefaultWidth returns the default glyph width (1000 EM-units).
func (f *Font) DefaultWidth() int {
	if f.UnitsPerEm == 0 || len(f.RawData) < 12 {
		return ttfUPEmInt
	}
	if hmtxTable, err := findTable(f.RawData, "hmtx"); err == nil && len(hmtxTable) >= ttfDWordSize {
		w := int(binary.BigEndian.Uint16(hmtxTable[0:]))
		if w > 0 {
			return int(float64(w)*ttfUPEm/float64(f.UnitsPerEm) + roundingHalf)
		}
	}
	return ttfUPEmInt
}

// UsedRunes returns all runes that have been added to the font, sorted in ascending order.
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
