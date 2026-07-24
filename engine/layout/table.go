package layout

import "fmt"

type CellStyle struct {
	FontName    string
	FontSize    float64
	TextColor   [3]float64
	FillColor   *[3]float64
	Border      *BorderStyle
	BorderLeft  *BorderStyle
	BorderRight *BorderStyle
	BorderTop   *BorderStyle
	BorderBottom *BorderStyle
	Padding     float64
	Align       Alignment
}

type Cell struct {
	Text  string
	Style CellStyle
	W, H  float64
}

type Row struct {
	Cells  []Cell
	Height float64
}

type TableLayout struct {
	ColWidths []float64
	Rows      []Row
}

// LayoutResult holds the builders and the final y position after laying out rows.
type LayoutResult struct {
	Builders []*ContentBuilder
	Y        float64
}

func (tl *TableLayout) LayOut(marginLeft, marginTop, pageW, pageH float64, startCB *ContentBuilder) (LayoutResult, error) {
	contentBottom := marginTop
	return tl.layOutFrom(marginLeft, marginTop, pageW, pageH, pageH-marginTop, startCB, contentBottom)
}

// LayOutFrom continues laying out rows starting from a given y position.
// Useful when chaining multiple TableLayouts on the same content builder.
func (tl *TableLayout) LayOutFrom(marginLeft, marginTop, pageW, pageH, y float64, startCB *ContentBuilder) (LayoutResult, error) {
	contentBottom := marginTop
	return tl.layOutFrom(marginLeft, marginTop, pageW, pageH, y, startCB, contentBottom)
}

func (tl *TableLayout) layOutFrom(marginLeft, marginTop, pageW, pageH, y float64, startCB *ContentBuilder, contentBottom float64) (LayoutResult, error) {
	builders := []*ContentBuilder{startCB}
	cb := startCB

	for _, row := range tl.Rows {
		if y-row.Height < contentBottom {
			cb = NewContentBuilder(pageW, pageH)
			builders = append(builders, cb)
			y = pageH - marginTop
		}

		x := marginLeft
		for ci, cell := range row.Cells {
			var cellW float64
			if ci < len(tl.ColWidths) {
				cellW = tl.ColWidths[ci]
			} else if cell.W > 0 {
				cellW = cell.W
			} else {
				cellW = 50
			}

			cellRect := Rect{X: x, Y: y - row.Height, W: cellW, H: row.Height}

			if cell.Style.FillColor != nil {
				cb.DrawRect(cellRect, cell.Style.FillColor, nil)
			}

			drawSide(cb, cellRect, cell.Style.BorderLeft, "left")
			drawSide(cb, cellRect, cell.Style.BorderRight, "right")
			drawSide(cb, cellRect, cell.Style.BorderTop, "top")
			drawSide(cb, cellRect, cell.Style.BorderBottom, "bottom")
			if cell.Style.Border != nil {
				cb.DrawRect(cellRect, nil, cell.Style.Border)
			}

			availW := cellW - cell.Style.Padding*2
			if availW < 1 {
				availW = 1
			}
			lines := WrapText(cell.Text, cell.Style.FontSize, availW)

			for li, line := range lines {
				tx := x + cell.Style.Padding
				switch cell.Style.Align {
				case AlignCenter:
					tx = x + cellW/2 - textWidth(line, cell.Style.FontSize)/2
				case AlignRight:
					tx = x + cellW - textWidth(line, cell.Style.FontSize) - cell.Style.Padding
				}
				lineY := y - row.Height + cell.Style.Padding + float64(li)*cell.Style.FontSize*1.2

				textRun := TextRun{
					Text:     line,
					FontName: cell.Style.FontName,
					FontSize: cell.Style.FontSize,
					Color:    cell.Style.TextColor,
					X:        tx,
					Y:        lineY,
				}
				if textRun.Color == [3]float64{0, 0, 0} {
					textRun.Color = cell.Style.TextColor
				}
				cb.PlaceText(textRun)
			}

			x += cellW
		}

		y -= row.Height
	}

	return LayoutResult{Builders: builders, Y: y}, nil
}

func drawSide(cb *ContentBuilder, r Rect, bs *BorderStyle, side string) {
	if bs == nil {
		return
	}
	fmt.Fprintf(&cb.Stream.Buf, "%s %s %s RG\n", fmtFloat(bs.Color[0]), fmtFloat(bs.Color[1]), fmtFloat(bs.Color[2]))
	fmt.Fprintf(&cb.Stream.Buf, "%s w\n", fmtFloat(bs.Width))
	var x1, y1, x2, y2 float64
	switch side {
	case "left":
		x1, y1 = r.X, r.Y
		x2, y2 = r.X, r.Y+r.H
	case "right":
		x1, y1 = r.X+r.W, r.Y
		x2, y2 = r.X+r.W, r.Y+r.H
	case "top":
		x1, y1 = r.X, r.Y+r.H
		x2, y2 = r.X+r.W, r.Y+r.H
	case "bottom":
		x1, y1 = r.X, r.Y
		x2, y2 = r.X+r.W, r.Y
	}
	fmt.Fprintf(&cb.Stream.Buf, "%s %s m %s %s l S\n", fmtFloat(x1), fmtFloat(y1), fmtFloat(x2), fmtFloat(y2))
}
