package layout

import "github.com/chinmay/gocorepdfengine/engine/color"

const defaultCellPadding = 3

// CellStyleFromColors builds a CellStyle with fill and text colors.
func CellStyleFromColors(font string, size float64, fg color.RGB, bg *color.RGB, pad float64) CellStyle {
	cs := CellStyle{
		FontName:  font,
		FontSize:  size,
		TextColor: [3]float64{fg[0], fg[1], fg[2]},
		Padding:   pad,
	}
	if bg != nil {
		c := [3]float64{bg[0], bg[1], bg[2]}
		cs.FillColor = &c
	}
	return cs
}

// DefaultBorder is a thin black border for table cells.
func DefaultBorder() *BorderStyle {
	return &BorderStyle{Width: half, Color: [3]float64{0.6, 0.6, 0.6}}
}

// StyledCell is a convenience constructor.
func StyledCell(text string, font string, size float64, fg color.RGB, bg *color.RGB, w, h float64) Cell {
	cs := CellStyleFromColors(font, size, fg, bg, defaultCellPadding)
	cs.Border = DefaultBorder()
	return Cell{Text: text, Style: cs, W: w, H: h}
}
