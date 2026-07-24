package content

import (
	"bytes"
	"compress/flate"
	"fmt"
	"strconv"
)

type Stream struct {
	Buf        bytes.Buffer
	Compressed bool
}

func NewStream() *Stream {
	return &Stream{}
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func (s *Stream) BT() {
	s.Buf.WriteString("BT\n")
}

func (s *Stream) ET() {
	s.Buf.WriteString("ET\n")
}

func (s *Stream) Tf(fontName string, size float64) {
	fmt.Fprintf(&s.Buf, "/%s %s Tf\n", fontName, fmtFloat(size))
}

func (s *Stream) Td(tx, ty float64) {
	fmt.Fprintf(&s.Buf, "%s %s Td\n", fmtFloat(tx), fmtFloat(ty))
}

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
				fmt.Fprintf(&s.Buf, "\\%03o", r)
			} else {
				s.Buf.WriteRune(r)
			}
		}
	}
	s.Buf.WriteString(") Tj\n")
}

func (s *Stream) TjCID(text string) {
	s.Buf.WriteByte('<')
	for _, r := range text {
		fmt.Fprintf(&s.Buf, "%04X", r)
	}
	s.Buf.WriteString("> Tj\n")
}

func (s *Stream) Tm(a, b, c, d, e, f float64) {
	fmt.Fprintf(&s.Buf, "%s %s %s %s %s %s Tm\n",
		fmtFloat(a), fmtFloat(b), fmtFloat(c),
		fmtFloat(d), fmtFloat(e), fmtFloat(f))
}

func (s *Stream) TL(leading float64) {
	fmt.Fprintf(&s.Buf, "%s TL\n", fmtFloat(leading))
}

func (s *Stream) re(x, y, w, h float64) {
	fmt.Fprintf(&s.Buf, "%s %s %s %s re\n",
		fmtFloat(x), fmtFloat(y), fmtFloat(w), fmtFloat(h))
}

func (s *Stream) m(x, y float64) {
	fmt.Fprintf(&s.Buf, "%s %s m\n", fmtFloat(x), fmtFloat(y))
}

func (s *Stream) l(x, y float64) {
	fmt.Fprintf(&s.Buf, "%s %s l\n", fmtFloat(x), fmtFloat(y))
}

func (s *Stream) h() {
	s.Buf.WriteString("h\n")
}

func (s *Stream) S() {
	s.Buf.WriteString("S\n")
}

func (s *Stream) f() {
	s.Buf.WriteString("f\n")
}

func (s *Stream) s() {
	s.Buf.WriteString("s\n")
}

func (s *Stream) w(width float64) {
	fmt.Fprintf(&s.Buf, "%s w\n", fmtFloat(width))
}

func (s *Stream) J(capStyle int) {
	fmt.Fprintf(&s.Buf, "%d J\n", capStyle)
}

func (s *Stream) j(joinStyle int) {
	fmt.Fprintf(&s.Buf, "%d j\n", joinStyle)
}

func (s *Stream) RG(r, g, b float64) {
	fmt.Fprintf(&s.Buf, "%s %s %s RG\n", fmtFloat(r), fmtFloat(g), fmtFloat(b))
}

func (s *Stream) rg(r, g, b float64) {
	fmt.Fprintf(&s.Buf, "%s %s %s rg\n", fmtFloat(r), fmtFloat(g), fmtFloat(b))
}

func (s *Stream) Do(name string) {
	fmt.Fprintf(&s.Buf, "/%s Do\n", name)
}

func (s *Stream) cm(a, b, c, d, e, f float64) {
	fmt.Fprintf(&s.Buf, "%s %s %s %s %s %s cm\n",
		fmtFloat(a), fmtFloat(b), fmtFloat(c),
		fmtFloat(d), fmtFloat(e), fmtFloat(f))
}

func (s *Stream) q() {
	s.Buf.WriteString("q\n")
}

func (s *Stream) Q() {
	s.Buf.WriteString("Q\n")
}

func (s *Stream) BDC(tag string, mcid int) {
	fmt.Fprintf(&s.Buf, "/%s <</MCID %d>> BDC\n", tag, mcid)
}

func (s *Stream) BMC(tag string) {
	fmt.Fprintf(&s.Buf, "/%s BMC\n", tag)
}

func (s *Stream) EMC() {
	s.Buf.WriteString("EMC\n")
}

func (s *Stream) ArtifactBMC(attached string) {
	if attached != "" {
		fmt.Fprintf(&s.Buf, "/Artifact <</Attached /%s>> BMC\n", attached)
	} else {
		s.Buf.WriteString("/Artifact BMC\n")
	}
}

func (s *Stream) Bytes() []byte {
	return s.Buf.Bytes()
}

func (s *Stream) Compress() error {
	var compressed bytes.Buffer
	w, err := flate.NewWriter(&compressed, flate.DefaultCompression)
	if err != nil {
		return err
	}
	if _, err := w.Write(s.Buf.Bytes()); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	s.Buf = compressed
	s.Compressed = true
	return nil
}
