package layout

import (
	"strings"
	"testing"
)

func TestContentBuilder(t *testing.T) {
	cb := NewContentBuilder(612, 792)
	cb.PlaceText(TextRun{
		Text:     "Hello World",
		FontName: "Helvetica",
		FontSize: 12,
		Color:    [3]float64{0, 0, 0},
		X:        100,
		Y:        700,
	})

	data := cb.Bytes()
	output := string(data)
	if !strings.Contains(output, "BT") {
		t.Error("expected content stream to contain BT")
	}
	if !strings.Contains(output, "Tj") {
		t.Error("expected content stream to contain Tj operator")
	}
	if !strings.Contains(output, "<00480065006C006C006F00200057006F0072006C0064>") {
		t.Error("expected content stream to contain hex-encoded text")
	}
}

func TestContentBuilder_FontMapping(t *testing.T) {
	cb := NewContentBuilder(612, 792)
	cb.PlaceText(TextRun{
		Text:     "A",
		FontName: "Helvetica",
		FontSize: 12,
		Color:    [3]float64{0, 0, 0},
		X:        50,
		Y:        700,
	})
	cb.PlaceText(TextRun{
		Text:     "B",
		FontName: "Times",
		FontSize: 10,
		Color:    [3]float64{0, 0, 0},
		X:        150,
		Y:        700,
	})

	if cb.FontRes["Helvetica"] != "F1" {
		t.Errorf("expected Helvetica -> F1, got %s", cb.FontRes["Helvetica"])
	}
	if cb.FontRes["Times"] != "F2" {
		t.Errorf("expected Times -> F2, got %s", cb.FontRes["Times"])
	}
}

func TestDrawRect_Fill(t *testing.T) {
	cb := NewContentBuilder(612, 792)
	fill := [3]float64{0.5, 0.5, 0.5}
	cb.DrawRect(Rect{X: 10, Y: 10, W: 100, H: 50}, &fill, nil)

	data := cb.Bytes()
	output := string(data)
	if !strings.Contains(output, "rg") {
		t.Error("expected rg operator in fill output")
	}
	if !strings.Contains(output, "re") {
		t.Error("expected re operator in fill output")
	}
	if !strings.Contains(output, "f") {
		t.Error("expected f operator in fill output")
	}
}

func TestDrawRect_Border(t *testing.T) {
	cb := NewContentBuilder(612, 792)
	border := &BorderStyle{Width: 1, Color: [3]float64{0, 0, 0}}
	cb.DrawRect(Rect{X: 10, Y: 10, W: 100, H: 50}, nil, border)

	data := cb.Bytes()
	output := string(data)
	if !strings.Contains(output, "RG") {
		t.Error("expected RG operator in border output")
	}
	if !strings.Contains(output, "w") {
		t.Error("expected w operator in border output")
	}
	if !strings.Contains(output, "S") {
		t.Error("expected S operator in border output")
	}
}

func TestTableLayout(t *testing.T) {
	cb := NewContentBuilder(612, 792)

	tl := &TableLayout{
		ColWidths: []float64{100, 100, 100},
		Rows: []Row{
			{
				Cells: []Cell{
					{Text: "A1", Style: CellStyle{FontName: "Helvetica", FontSize: 10, TextColor: [3]float64{0, 0, 0}, Padding: 4}, W: 100, H: 30},
					{Text: "B1", Style: CellStyle{FontName: "Helvetica", FontSize: 10, TextColor: [3]float64{0, 0, 0}, Padding: 4}, W: 100, H: 30},
					{Text: "C1", Style: CellStyle{FontName: "Helvetica", FontSize: 10, TextColor: [3]float64{0, 0, 0}, Padding: 4}, W: 100, H: 30},
				},
				Height: 30,
			},
			{
				Cells: []Cell{
					{Text: "A2", Style: CellStyle{FontName: "Helvetica", FontSize: 10, TextColor: [3]float64{0, 0, 0}, Padding: 4}, W: 100, H: 30},
					{Text: "B2", Style: CellStyle{FontName: "Helvetica", FontSize: 10, TextColor: [3]float64{0, 0, 0}, Padding: 4}, W: 100, H: 30},
					{Text: "C2", Style: CellStyle{FontName: "Helvetica", FontSize: 10, TextColor: [3]float64{0, 0, 0}, Padding: 4}, W: 100, H: 30},
				},
				Height: 30,
			},
		},
	}

	builders, err := tl.LayOut(50, 50, 612, 792, cb)
	if err != nil {
		t.Fatal(err)
	}
	if len(builders) < 1 {
		t.Error("expected at least one ContentBuilder")
	}
}

func TestTableLayout_PageBreak(t *testing.T) {
	cb := NewContentBuilder(200, 100)

	rows := make([]Row, 5)
	for i := range rows {
		rows[i] = Row{
			Cells: []Cell{
				{Text: "X", Style: CellStyle{FontName: "Helvetica", FontSize: 10, TextColor: [3]float64{0, 0, 0}, Padding: 2}, W: 50, H: 30},
			},
			Height: 30,
		}
	}

	tl := &TableLayout{
		ColWidths: []float64{50},
		Rows:      rows,
	}

	builders, err := tl.LayOut(10, 10, 200, 100, cb)
	if err != nil {
		t.Fatal(err)
	}
	if len(builders) < 2 {
		t.Error("expected page break to produce multiple builders with small page height")
	}
}
