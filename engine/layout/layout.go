package layout

import (
	"fmt"
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/content"
	"github.com/chinmay/gocorepdfengine/engine/image"
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
	Stream       *content.Stream
	FontRes      map[string]string
	UsedFonts    map[string]bool
	ImageObjects map[string]*ImageObj
	MCID         int
	Width, Height float64
}

type ImageObj struct {
	Img   *image.Image
	Data  []byte
}

func NewContentBuilder(width, height float64) *ContentBuilder {
	return &ContentBuilder{
		Stream:       content.NewStream(),
		FontRes:      make(map[string]string),
		UsedFonts:    make(map[string]bool),
		ImageObjects: make(map[string]*ImageObj),
		MCID:         0,
		Width:        width,
		Height:       height,
	}
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func textWidth(text string, fontSize float64) float64 {
	return float64(len(text)) * fontSize * 0.52
}

func WrapText(text string, fontSize, maxWidth float64) []string {
	if maxWidth <= 0 || textWidth(text, fontSize) <= maxWidth {
		return []string{text}
	}
	var lines []string
	runes := []rune(text)
	start := 0
	for start < len(runes) {
		if len(runes)-start == 1 {
			lines = append(lines, string(runes[start:]))
			break
		}
		// Find the longest initial segment that fits.
		end := start + 1
		for end <= len(runes) && textWidth(string(runes[start:end]), fontSize) <= maxWidth {
			end++
		}
		// If the whole remaining text fits, use it.
		if end > len(runes) {
			lines = append(lines, string(runes[start:]))
			break
		}
		// Try to break at a space (word boundary, trimming the trailing space).
		breakAt := end - 1
		for i := end - 1; i > start; i-- {
			if runes[i] == ' ' {
				breakAt = i
				break
			}
		}
		if breakAt == start {
			// No space found — break at character boundary.
			breakAt = end - 1
		}
		lines = append(lines, string(runes[start:breakAt]))
		start = breakAt
		// Skip leading space on the next line.
		for start < len(runes) && runes[start] == ' ' {
			start++
		}
	}
	return lines
}

func (cb *ContentBuilder) PlaceText(run TextRun) {
	if run.Text == "" {
		return
	}
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
	cb.Stream.TjCID(run.Text)
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

func (cb *ContentBuilder) PlaceWatermark(text string, pageW, pageH float64) {
	label, ok := cb.FontRes["Helvetica"]
	if !ok {
		label = fmt.Sprintf("F%d", len(cb.FontRes)+1)
		cb.FontRes["Helvetica"] = label
		cb.UsedFonts["Helvetica"] = true
	}
	cosA := 0.71
	sinA := 0.71
	fmt.Fprintf(&cb.Stream.Buf, "/Artifact <</Attached [/Top] /Type /Pagination >> BDC\n")
	fmt.Fprintf(&cb.Stream.Buf, "q\n")
	fmt.Fprintf(&cb.Stream.Buf, "%s %s %s rg %s %s %s RG\n",
		fmtFloat(0.85), fmtFloat(0.85), fmtFloat(0.85),
		fmtFloat(0.85), fmtFloat(0.85), fmtFloat(0.85))
	fmt.Fprintf(&cb.Stream.Buf, "BT\n")
	fmt.Fprintf(&cb.Stream.Buf, "/%s 74 Tf\n", label)
	fmt.Fprintf(&cb.Stream.Buf, "%s %s %s %s %s %s Tm\n",
		fmtFloat(cosA), fmtFloat(sinA), fmtFloat(-sinA), fmtFloat(cosA),
		fmtFloat(pageW*0.2), fmtFloat(pageH*0.3))
	fmt.Fprintf(&cb.Stream.Buf, "(%s) Tj\n", text)
	fmt.Fprintf(&cb.Stream.Buf, "ET\n")
	fmt.Fprintf(&cb.Stream.Buf, "Q\n")
	fmt.Fprintf(&cb.Stream.Buf, "EMC\n")
}

func (cb *ContentBuilder) PlaceImage(img *image.Image, objName string, x, y, w, h float64) {
	cb.ImageObjects[objName] = &ImageObj{Img: img, Data: img.Data}
	// Preserve aspect ratio: scale to fit within w×h, then center.
	iw, ih := float64(img.Width), float64(img.Height)
	scale := w / iw
	if ih*scale > h {
		scale = h / ih
	}
	dw := iw * scale
	dh := ih * scale
	dx := x + (w-dw)/2
	dy := y + (h-dh)/2
	fmt.Fprintf(&cb.Stream.Buf, "q\n")
	fmt.Fprintf(&cb.Stream.Buf, "%s 0 0 %s %s %s cm\n", fmtFloat(dw), fmtFloat(dh), fmtFloat(dx), fmtFloat(dy))
	cb.Stream.Do(objName)
	fmt.Fprintf(&cb.Stream.Buf, "Q\n")
}

func (cb *ContentBuilder) Bytes() []byte {
	return cb.Stream.Bytes()
}
