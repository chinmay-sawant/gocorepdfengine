package layout

import (
	"fmt"
	"strconv"
	"strings"
)

type Alignment int

const (
	AlignLeft   Alignment = 0
	AlignCenter Alignment = 1
	AlignRight  Alignment = 2
)

type CellBorder [4]bool // left, right, top, bottom

type CellProps struct {
	FontName  string
	FontSize  float64
	Bold      bool
	Italic    bool
	Underline bool
	Align     Alignment
	Border    CellBorder
}

func ParseProps(s string) (CellProps, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 8 {
		return CellProps{}, fmt.Errorf("props needs 8 colon-separated fields, got %d in %q", len(parts), s)
	}

	p := CellProps{FontName: parts[0]}

	size, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return CellProps{}, fmt.Errorf("props fontsize: %w", err)
	}
	p.FontSize = size

	if len(parts[2]) == 3 {
		p.Bold = parts[2][0] == '1'
		p.Italic = parts[2][1] == '1'
		p.Underline = parts[2][2] == '1'
	}

	switch parts[3] {
	case "left":
		p.Align = AlignLeft
	case "center":
		p.Align = AlignCenter
	case "right":
		p.Align = AlignRight
	default:
		p.Align = AlignLeft
	}

	for i := 0; i < 4; i++ {
		p.Border[i] = parts[4+i] == "1"
	}

	return p, nil
}
