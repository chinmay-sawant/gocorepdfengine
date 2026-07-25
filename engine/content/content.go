// Package content provides PDF content stream building primitives.
package content

import (
	"bytes"
	"compress/flate"
	"fmt"
	"strconv"
	"sync"
)

var flateWriterPool = sync.Pool{
	New: func() any { // returns *flate.Writer
		w, _ := flate.NewWriter(nil, flate.BestSpeed)
		return w
	},
}

type Stream struct {
	Buf        bytes.Buffer
	Compressed bool
}

// NewStream returns a new empty Stream.
func NewStream() *Stream {
	return &Stream{}
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// BT begins a text object.
func (s *Stream) BT() {
	s.Buf.WriteString("BT\n")
}

// ET ends a text object.
func (s *Stream) ET() {
	s.Buf.WriteString("ET\n")
}

// Tf sets the font and font size.
func (s *Stream) Tf(fontName string, size float64) {
	fmt.Fprintf(&s.Buf, "/%s %s Tf\n", fontName, fmtFloat(size))
}

// Td moves the text position.
func (s *Stream) Td(tx, ty float64) {
	fmt.Fprintf(&s.Buf, "%s %s Td\n", fmtFloat(tx), fmtFloat(ty))
}

// Tj writes a text string using PDF literal string encoding.
func (s *Stream) Tj(text string) {
	s.Buf.WriteByte('(')
	for _, r := range text {
		switch r {
		case '(':
			s.Buf.WriteString("\\(")
		case ')':
			s.Buf.WriteString("\\)")
		case '\\':
			s.Buf.WriteString("\\\\")
		case '\n':
			s.Buf.WriteString("\\n")
		case '\r':
			s.Buf.WriteString("\\r")
		case '\t':
			s.Buf.WriteString("\\t")
		default:
			if r >= 128 {
				s.Buf.WriteByte('\\')
				s.Buf.WriteByte('0' + byte(r>>6))
				s.Buf.WriteByte('0' + byte((r>>3)&7))
				s.Buf.WriteByte('0' + byte(r&7))
			} else {
				s.Buf.WriteRune(r)
			}
		}
	}
	s.Buf.WriteString(") Tj\n")
}

// TjCID writes a text string using PDF CID (hex) encoding.
func (s *Stream) TjCID(text string) {
	const hex = "0123456789ABCDEF"
	s.Buf.WriteByte('<')
	for _, r := range text {
		s.Buf.WriteByte(hex[(r>>12)&0xF])
		s.Buf.WriteByte(hex[(r>>8)&0xF])
		s.Buf.WriteByte(hex[(r>>4)&0xF])
		s.Buf.WriteByte(hex[r&0xF])
	}
	s.Buf.WriteString("> Tj\n")
}

// Tm sets the text matrix.
func (s *Stream) Tm(a, b, c, d, e, f float64) {
	fmt.Fprintf(&s.Buf, "%s %s %s %s %s %s Tm\n",
		fmtFloat(a), fmtFloat(b), fmtFloat(c),
		fmtFloat(d), fmtFloat(e), fmtFloat(f))
}

// TL sets the text leading.
func (s *Stream) TL(leading float64) {
	fmt.Fprintf(&s.Buf, "%s TL\n", fmtFloat(leading))
}

// S strokes the path.
func (s *Stream) S() {
	s.Buf.WriteString("S\n")
}

// J sets the line cap style.
func (s *Stream) J(capStyle int) {
	fmt.Fprintf(&s.Buf, "%d J\n", capStyle)
}

// RG sets the stroke color (RGB).
func (s *Stream) RG(r, g, b float64) {
	fmt.Fprintf(&s.Buf, "%s %s %s RG\n", fmtFloat(r), fmtFloat(g), fmtFloat(b))
}

// Do paints a named XObject.
func (s *Stream) Do(name string) {
	fmt.Fprintf(&s.Buf, "/%s Do\n", name)
}

// Q restores the graphics state.
func (s *Stream) Q() {
	s.Buf.WriteString("Q\n")
}

// BDC begins a marked-content sequence with a property list.
func (s *Stream) BDC(tag string, mcid int) {
	fmt.Fprintf(&s.Buf, "/%s <</MCID %d>> BDC\n", tag, mcid)
}

// BMC begins a marked-content sequence.
func (s *Stream) BMC(tag string) {
	fmt.Fprintf(&s.Buf, "/%s BMC\n", tag)
}

// EMC ends a marked-content sequence.
func (s *Stream) EMC() {
	s.Buf.WriteString("EMC\n")
}

// ArtifactBMC begins an Artifact marked-content sequence.
func (s *Stream) ArtifactBMC(attached string) {
	if attached != "" {
		fmt.Fprintf(&s.Buf, "/Artifact <</Attached /%s>> BMC\n", attached)
	} else {
		s.Buf.WriteString("/Artifact BMC\n")
	}
}

// Bytes returns the uncompressed content bytes.
func (s *Stream) Bytes() []byte {
	return s.Buf.Bytes()
}

// Compress compresses the stream content using Flate.
func (s *Stream) Compress() error {
	var compressed bytes.Buffer
	w := flateWriterPool.Get().(*flate.Writer)
	w.Reset(&compressed)
	if _, err := w.Write(s.Buf.Bytes()); err != nil { // cold path (error)
		flateWriterPool.Put(w)
		return fmt.Errorf("content: compress: %w", err) // cold path (error)
	}
	if err := w.Close(); err != nil { // cold path (error)
		flateWriterPool.Put(w)
		return fmt.Errorf("content: compress: %w", err) // cold path (error)
	}
	flateWriterPool.Put(w)
	s.Buf = compressed
	s.Compressed = true
	return nil
}
