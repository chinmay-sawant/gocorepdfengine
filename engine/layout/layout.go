package layout

import (
	"fmt"
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/content"
)

type Point struct {
	X, Y float64
}

type Rect struct {
	X, Y, W, H float64
}

type TextRun struct {
	Text     string
	FontName string
	FontSize float64
	Color    [3]float64
	X, Y     float64
}

type BorderStyle struct {
	Width float64
	Color [3]float64
}

type ContentBuilder struct {
	Stream    *content.Stream
	FontRes   map[string]string
	UsedFonts map[string]bool
	MCID      int
	Width, Height float64
}

func NewContentBuilder(width, height float64) *ContentBuilder {
	return &ContentBuilder{
		Stream:    content.NewStream(),
		FontRes:   make(map[string]string),
		UsedFonts: make(map[string]bool),
		MCID:      0,
		Width:     width,
		Height:    height,
	}
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (cb *ContentBuilder) PlaceText(run TextRun) {
	label, ok := cb.FontRes[run.FontName]
	if !ok {
		label = fmt.Sprintf("F%d", len(cb.FontRes)+1)
		cb.FontRes[run.FontName] = label
		cb.UsedFonts[run.FontName] = true
	}

	fmt.Fprintf(&cb.Stream.Buf, "%s %s %s rg\n", fmtFloat(run.Color[0]), fmtFloat(run.Color[1]), fmtFloat(run.Color[2]))
	cb.Stream.BT()
	cb.Stream.Tf(label, run.FontSize)
	cb.Stream.Td(run.X, run.Y)
	cb.Stream.Tj(run.Text)
	cb.Stream.ET()
}

func (cb *ContentBuilder) DrawRect(r Rect, fill *[3]float64, border *BorderStyle) {
	if fill != nil {
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s rg\n", fmtFloat(fill[0]), fmtFloat(fill[1]), fmtFloat(fill[2]))
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s %s re\n", fmtFloat(r.X), fmtFloat(r.Y), fmtFloat(r.W), fmtFloat(r.H))
		cb.Stream.Buf.WriteString("f\n")
	}
	if border != nil {
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s RG\n", fmtFloat(border.Color[0]), fmtFloat(border.Color[1]), fmtFloat(border.Color[2]))
		fmt.Fprintf(&cb.Stream.Buf, "%s w\n", fmtFloat(border.Width))
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s %s re\n", fmtFloat(r.X), fmtFloat(r.Y), fmtFloat(r.W), fmtFloat(r.H))
		cb.Stream.Buf.WriteString("S\n")
	}
}

func (cb *ContentBuilder) Bytes() []byte {
	return cb.Stream.Bytes()
}
