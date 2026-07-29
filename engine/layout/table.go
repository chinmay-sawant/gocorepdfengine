package layout

import (
	"fmt"
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/image"
)

// codehound-ignore: BP-40
const (
	marginMultiplier = 2
	pointToPixel     = 72
	decimalBase      = 10
	lineHeightFactor = 1.2
)

// CellStyle controls the visual appearance of a table cell.
type CellStyle struct {
	FontName     string
	FontSize     float64
	TextColor    [3]float64
	FillColor    *[3]float64
	Border       *BorderStyle
	BorderLeft   *BorderStyle
	BorderRight  *BorderStyle
	BorderTop    *BorderStyle
	BorderBottom *BorderStyle
	Padding      float64
	Align        Alignment
}

// CellImage holds raw image data to be placed inside a cell.
type CellImage struct {
	Data   []byte // raw PNG or JPEG bytes
	IsJPEG bool
}

// Cell is a single table cell with text, style, dimensions, and optional image.
type Cell struct {
	Text  string
	Style CellStyle
	W, H  float64
	Image *CellImage
}

// Row is a horizontal collection of cells with a fixed height.
type Row struct {
	Cells  []Cell
	Height float64
}

// TableLayout holds column widths and rows for paginated table layout.
type TableLayout struct {
	ColWidths []float64
	Rows      []Row
}

// Result holds the builders and the final y position after laying out rows.
type Result struct {
	Builders []*ContentBuilder
	Y        float64
}

// LayOut lays out all rows starting from marginTop, returning the builders
// and the final Y position. New pages are created automatically when content
// exceeds the available height.
func (tl *TableLayout) LayOut(marginLeft, marginTop, pageW, pageH float64, startCB *ContentBuilder) (Result, error) {
	contentBottom := marginTop
	return tl.layOutFrom(marginLeft, marginTop, pageW, pageH, pageH-marginTop, startCB, contentBottom)
}

// LayOutFrom continues laying out rows starting from a given y position.
// Useful when chaining multiple TableLayouts on the same content builder.
func (tl *TableLayout) LayOutFrom(marginLeft, marginTop, pageW, pageH, y float64, startCB *ContentBuilder) (Result, error) {
	contentBottom := marginTop
	return tl.layOutFrom(marginLeft, marginTop, pageW, pageH, y, startCB, contentBottom)
}

func (tl *TableLayout) layOutFrom(marginLeft, marginTop, pageW, pageH, y float64, startCB *ContentBuilder, contentBottom float64) (Result, error) {
	builders := []*ContentBuilder{startCB}
	cb := startCB

	// Compute available content width from margins and page width.
	contentW := pageW - marginLeft*marginMultiplier
	if contentW <= 0 {
		contentW = pageW - pointToPixel
	}

	var cellWidthsBuf = make([]float64, 0, 8)
	var imgBuf []byte
	for _, row := range tl.Rows {
		cellsLen := len(row.Cells)
		if y-row.Height < contentBottom {
			cb = NewContentBuilder(pageW, pageH)
			builders = append(builders, cb)
			y = pageH - marginTop
		}

		// Pre-compute effective cell widths for this row, ensuring total = contentW.
		var cellWidths []float64
		if cap(cellWidthsBuf) >= cellsLen {
			cellWidths = cellWidthsBuf[:cellsLen]
			for i := range cellWidths {
				cellWidths[i] = 0
			}
		} else {
			// codehound-ignore: PERF-3
			cellWidthsBuf = make([]float64, cellsLen)
			cellWidths = cellWidthsBuf
		}
		var explicitSum float64
		var explicitCount int
		for ci, cell := range row.Cells {
			if cell.W > 0 {
				cellWidths[ci] = cell.W
				explicitSum += cell.W
				explicitCount++
			}
		}
		switch {
		case explicitCount == cellsLen && explicitSum > 0:
			// All cells have explicit widths — scale to fill contentW.
			scale := contentW / explicitSum
			for ci := range cellWidths {
				cellWidths[ci] *= scale
			}
		case explicitCount > 0 && explicitSum < contentW:
			// Some cells have explicit widths — distribute remaining space equally.
			remaining := contentW - explicitSum
			implicitCount := cellsLen - explicitCount
			share := remaining / float64(implicitCount)
			for ci := range cellWidths {
				if cellWidths[ci] <= 0 {
					cellWidths[ci] = share
				}
			}
		default:
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
				imgBuf = strconv.AppendInt(imgBuf[:0], int64(len(cb.ImageObjects)+1), decimalBase)
				imgName := "Img" + string(imgBuf)
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

			// codehound-ignore: PERF-230
			tw := cellTextWidth(cell.Text, cell.Style.FontSize)
			tx := x + cell.Style.Padding
			switch cell.Style.Align {
			case AlignLeft:
			case AlignCenter:
				tx = x + cellW*half - tw*half
			case AlignRight:
				tx = x + cellW - tw - cell.Style.Padding
			}
			startY := y - row.Height + (row.Height-cell.Style.FontSize*lineHeightFactor)*half

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

	return Result{Builders: builders, Y: y}, nil
}

func drawSide(cb *ContentBuilder, r Rect, bs *BorderStyle, side string) {
	if bs == nil {
		return
	}
	b0 := fmtFloat(bs.Color[0])
	b1 := fmtFloat(bs.Color[1])
	b2 := fmtFloat(bs.Color[2])
	bw := fmtFloat(bs.Width)
	fmt.Fprintf(&cb.Stream.Buf, "%s %s %s RG\n", b0, b1, b2)
	fmt.Fprintf(&cb.Stream.Buf, "%s w\n", bw)
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
	rx1 := fmtFloat(x1)
	ry1 := fmtFloat(y1)
	rx2 := fmtFloat(x2)
	ry2 := fmtFloat(y2)
	fmt.Fprintf(&cb.Stream.Buf, "%s %s m %s %s l S\n", rx1, ry1, rx2, ry2)
}

func cellTextWidth(text string, fontSize float64) float64 {
	return textWidth(text, fontSize)
}
