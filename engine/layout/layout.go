// Package layout provides content-stream builders for positioning text, images,
// rectangles, and watermarks on a PDF page. It also offers text-wrapping and
// table-layout helpers used by higher-level rendering packages.
package layout

import (
	"fmt"
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/content"
	"github.com/chinmay/gocorepdfengine/engine/image"
)

// Point represents a 2D coordinate in PDF user-space units.
type Point struct {
	X, Y float64
}

// Rect represents a rectangle with lower-left corner (X, Y) and dimensions (W, H).
type Rect struct {
	X, Y, W, H float64
}

// TextRun holds all parameters needed to place a single line of text.
type TextRun struct {
	Text     string
	FontName string
	FontSize float64
	Color    [3]float64
	X, Y     float64
}

// BorderStyle describes a single-side border.
type BorderStyle struct {
	Width float64
	Color [3]float64
}

// ContentBuilder accumulates PDF content-stream operations and tracks
// font / image / marked-content identifiers used during layout.
type ContentBuilder struct {
	Stream       *content.Stream
	FontRes      map[string]string
	UsedFonts    map[string]bool
	ImageObjects map[string]*ImageObj
	MCID         int
	Width, Height float64
}

// ImageObj pairs a decoded image with its raw bytes for embedding.
type ImageObj struct {
	Img   *image.Image
	Data  []byte
}

// NewContentBuilder creates a ContentBuilder for a page of the given dimensions.
func NewContentBuilder(width, height float64) *ContentBuilder {
	return &ContentBuilder{
		Stream:       content.NewStream(),
		FontRes:      make(map[string]string),
		UsedFonts:    make(map[string]bool),
		ImageObjects: make(map[string]*ImageObj, 8), // BP-52: size hint for expected images
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

func charScale(fontSize float64) float64 {
	return fontSize * 0.52
}

// WrapText breaks text into lines that each fit within maxWidth at the given
// font size. Words are preserved when possible; otherwise characters are broken.
func WrapText(text string, fontSize, maxWidth float64) []string {
	scale := charScale(fontSize)
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
		for end <= len(runes) && float64(end-start)*scale <= maxWidth {
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

// PlaceText emits PDF content-stream operators to show a line of text.
func (cb *ContentBuilder) PlaceText(run TextRun) {
	if run.Text == "" {
		return
	}
	label, ok := cb.FontRes[run.FontName]
	if !ok {
		label = "F" + strconv.Itoa(len(cb.FontRes)+1)
		cb.FontRes[run.FontName] = label
		cb.UsedFonts[run.FontName] = true
	}

	r0 := fmtFloat(run.Color[0])
	r1 := fmtFloat(run.Color[1])
	r2 := fmtFloat(run.Color[2])
	fmt.Fprintf(&cb.Stream.Buf, "%s %s %s rg\n", r0, r1, r2)
	cb.Stream.BT()
	cb.Stream.Tf(label, run.FontSize)
	cb.Stream.Td(run.X, run.Y)
	cb.Stream.TjCID(run.Text)
	cb.Stream.ET()
}

// DrawRect fills and/or strokes a rectangle. Both fill and border are optional.
func (cb *ContentBuilder) DrawRect(r Rect, fill *[3]float64, border *BorderStyle) {
	rx := fmtFloat(r.X)
	ry := fmtFloat(r.Y)
	rw := fmtFloat(r.W)
	rh := fmtFloat(r.H)
	if fill != nil {
		f0 := fmtFloat(fill[0])
		f1 := fmtFloat(fill[1])
		f2 := fmtFloat(fill[2])
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s rg\n", f0, f1, f2)
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s %s re\n", rx, ry, rw, rh)
		cb.Stream.Buf.WriteString("f\n")
	}
	if border != nil {
		b0 := fmtFloat(border.Color[0])
		b1 := fmtFloat(border.Color[1])
		b2 := fmtFloat(border.Color[2])
		bw := fmtFloat(border.Width)
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s RG\n", b0, b1, b2)
		fmt.Fprintf(&cb.Stream.Buf, "%s w\n", bw)
		fmt.Fprintf(&cb.Stream.Buf, "%s %s %s %s re\n", rx, ry, rw, rh)
		cb.Stream.Buf.WriteString("S\n")
	}
}

// PlaceWatermark adds a diagonal "DRAFT"-style watermark as a PDF Artifact.
func (cb *ContentBuilder) PlaceWatermark(text string, pageW, pageH float64) {
	label, ok := cb.FontRes["Helvetica"]
	if !ok {
		label = "F" + strconv.Itoa(len(cb.FontRes)+1)
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

// PlaceImage adds a Do operator for an image XObject, scaling to fit w×h while
// preserving the aspect ratio and centering within the given rectangle.
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

// Bytes returns the accumulated PDF content-stream data.
func (cb *ContentBuilder) Bytes() []byte {
	return cb.Stream.Bytes()
}
