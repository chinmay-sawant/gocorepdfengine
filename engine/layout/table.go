package layout

import (
	"fmt"

	"github.com/chinmay/gocorepdfengine/engine/image"
)

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

type CellImage struct {
	Data        []byte // raw PNG or JPEG bytes
	IsJPEG      bool
}

type Cell struct {
	Text  string
	Style CellStyle
	W, H  float64
	Image *CellImage
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

	// Compute available content width from margins and page width.
	contentW := pageW - marginLeft*2
	if contentW <= 0 {
		contentW = pageW - 72
	}

	for _, row := range tl.Rows {
		if y-row.Height < contentBottom {
			cb = NewContentBuilder(pageW, pageH)
			builders = append(builders, cb)
			y = pageH - marginTop
		}

		// Pre-compute effective cell widths for this row, ensuring total = contentW.
		cellWidths := make([]float64, len(row.Cells))
		var explicitSum float64
		var explicitCount int
		for ci, cell := range row.Cells {
			if cell.W > 0 {
				cellWidths[ci] = cell.W
				explicitSum += cell.W
				explicitCount++
			}
		}
		if explicitCount == len(row.Cells) && explicitSum > 0 {
			// All cells have explicit widths — scale to fill contentW.
			scale := contentW / explicitSum
			for ci := range cellWidths {
				cellWidths[ci] *= scale
			}
		} else if explicitCount > 0 && explicitSum < contentW {
			// Some cells have explicit widths — distribute remaining space equally.
			remaining := contentW - explicitSum
			implicitCount := len(row.Cells) - explicitCount
			share := remaining / float64(implicitCount)
			for ci := range cellWidths {
				if cellWidths[ci] <= 0 {
					cellWidths[ci] = share
				}
			}
		} else {
			// No explicit widths — use base column widths.
			for ci := range cellWidths {
				if ci < len(tl.ColWidths) {
					cellWidths[ci] = tl.ColWidths[ci]
				} else {
					cellWidths[ci] = 50
				}
			}
		}

		x := marginLeft
		for ci, cell := range row.Cells {
			cellW := cellWidths[ci]

			cellRect := Rect{X: x, Y: y - row.Height, W: cellW, H: row.Height}

			if cell.Style.FillColor != nil {
				cb.DrawRect(cellRect, cell.Style.FillColor, nil)
			}

			// Render image if present (content before borders so borders stay on top).
			if cell.Image != nil {
				imgName := fmt.Sprintf("Img%d", len(cb.ImageObjects)+1)
				var img *image.Image
				var err error
				if cell.Image.IsJPEG {
					img, err = image.NewFromJPEG(cell.Image.Data)
				} else {
					img, err = image.NewFromPNG(cell.Image.Data)
				}
				if err == nil {
					cb.PlaceImage(img, imgName, cellRect.X, cellRect.Y, cellRect.W, cellRect.H)
				}
			}

			tx := x + cell.Style.Padding
			switch cell.Style.Align {
			case AlignCenter:
				tx = x + cellW/2 - textWidth(cell.Text, cell.Style.FontSize)/2
			case AlignRight:
				tx = x + cellW - textWidth(cell.Text, cell.Style.FontSize) - cell.Style.Padding
			}
			startY := y - row.Height + (row.Height-cell.Style.FontSize*1.2)/2

			textRun := TextRun{
				Text:     cell.Text,
				FontName: cell.Style.FontName,
				FontSize: cell.Style.FontSize,
				Color:    cell.Style.TextColor,
				X:        tx,
				Y:        startY,
			}
			if textRun.Color == [3]float64{0, 0, 0} {
				textRun.Color = cell.Style.TextColor
			}
			cb.PlaceText(textRun)

			// Borders on top so they overlay cell content (images, fills, text).
			drawSide(cb, cellRect, cell.Style.BorderLeft, "left")
			drawSide(cb, cellRect, cell.Style.BorderRight, "right")
			drawSide(cb, cellRect, cell.Style.BorderTop, "top")
			drawSide(cb, cellRect, cell.Style.BorderBottom, "bottom")
			if cell.Style.Border != nil {
				cb.DrawRect(cellRect, nil, cell.Style.Border)
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
