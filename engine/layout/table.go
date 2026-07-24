package layout

type CellStyle struct {
	FontName  string
	FontSize  float64
	TextColor [3]float64
	FillColor *[3]float64
	Border    *BorderStyle
	Padding   float64
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

func (tl *TableLayout) LayOut(marginLeft, marginTop, pageW, pageH float64, startCB *ContentBuilder) ([]*ContentBuilder, error) {
	builders := []*ContentBuilder{startCB}
	cb := startCB

	contentBottom := marginTop
	y := pageH - marginTop

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

			if cell.Style.Border != nil {
				cb.DrawRect(cellRect, nil, cell.Style.Border)
			}

			textRun := TextRun{
				Text:     cell.Text,
				FontName: cell.Style.FontName,
				FontSize: cell.Style.FontSize,
				Color:    cell.Style.TextColor,
				X:        x + cell.Style.Padding,
				Y:        y - row.Height + cell.Style.Padding,
			}
			if textRun.Color == [3]float64{0, 0, 0} {
				textRun.Color = cell.Style.TextColor
			}
			cb.PlaceText(textRun)

			x += cellW
		}

		y -= row.Height
	}

	return builders, nil
}
