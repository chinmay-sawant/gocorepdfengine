package write

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

type Encoder struct {
	buf bytes.Buffer
}

func NewEncoder() *Encoder {
	return &Encoder{}
}

func (e *Encoder) Write(p []byte) (int, error) {
	//nolint:wrapcheck
	return e.buf.Write(p)
}

func (e *Encoder) WriteString(s string) {
	e.buf.WriteString(s)
}

func (e *Encoder) Len() int {
	return e.buf.Len()
}

func (e *Encoder) Bytes() []byte {
	return e.buf.Bytes()
}

func (e *Encoder) WriteHeader() {
	e.buf.WriteString("%PDF-2.0\n")
	e.buf.WriteString("%\x80\x80\x80\x80\n")
}

func (e *Encoder) WriteComment(comment string) {
	fmt.Fprintf(&e.buf, "%% %s\n", comment)
}

func (e *Encoder) WriteDict(dict map[string]interface{}) {
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	e.buf.WriteString("<< ")
	for _, k := range keys {
		e.buf.WriteString(k)
		e.buf.WriteByte(' ')
		e.writeValue(dict[k])
		e.buf.WriteByte(' ')
	}
	e.buf.WriteString(">>")
}

// PDFString represents a PDF literal string value "(...)".
type PDFString string

func (e *Encoder) writeValue(v interface{}) {
	switch val := v.(type) {
	case string:
		e.buf.WriteString(val)
	case PDFString:
		e.buf.WriteString(StringLit(string(val)))
	case int:
		fmt.Fprintf(&e.buf, "%d", val)
	case int64:
		fmt.Fprintf(&e.buf, "%d", val)
	case float64:
		e.buf.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
	case bool:
		if val {
			e.buf.WriteString("true")
		} else {
			e.buf.WriteString("false")
		}
	case []interface{}:
		e.buf.WriteString("[ ")
		for _, item := range val {
			e.writeValue(item)
			e.buf.WriteByte(' ')
		}
		e.buf.WriteString("]")
	case map[string]interface{}:
		e.WriteDict(val)
	default:
		fmt.Fprintf(&e.buf, "%v", val)
	}
}

func (e *Encoder) WriteStream(dict map[string]interface{}, data []byte) {
	e.WriteDict(dict)
	e.buf.WriteString("\nstream\n")
	e.buf.Write(data)
	e.buf.WriteString("\nendstream")
}

func (e *Encoder) WriteXref(offsets []int64) {
	fmt.Fprintf(&e.buf, "xref\n0 %d\n", len(offsets))
	for i, off := range offsets {
		if i == 0 {
			fmt.Fprintf(&e.buf, "%010d %05d f \n", off, 65535) //nolint:mnd
		} else {
			fmt.Fprintf(&e.buf, "%010d %05d n \n", off, 0)
		}
	}
}

func (e *Encoder) WriteTrailer(size int, rootRef string, id []string) {
	e.buf.WriteString("trailer\n")
	dict := map[string]interface{}{
		"/Size": size,
		"/Root": rootRef,
	}
	if len(id) >= 2 {
		dict["/ID"] = []interface{}{id[0], id[1]}
	}
	e.WriteDict(dict)
	e.buf.WriteString("\n")
}

func (e *Encoder) WriteStartXref(offset int64) {
	fmt.Fprintf(&e.buf, "startxref\n%d\n", offset)
}

func (e *Encoder) WriteEOF() {
	e.buf.WriteString("%%EOF")
}

func Ref(objNum, genNum int) string {
	return fmt.Sprintf("%d %d R", objNum, genNum)
}

func Name(s string) string {
	return "/" + s
}

func StringLit(s string) string {
	var buf bytes.Buffer
	buf.WriteByte('(')
	for _, r := range s {
		switch r {
		case '(':
			buf.WriteString("\\(")
		case ')':
			buf.WriteString("\\)")
		case '\\':
			buf.WriteString("\\\\")
		case '\n':
			buf.WriteString("\\n")
		case '\r':
			buf.WriteString("\\r")
		case '\t':
			buf.WriteString("\\t")
		default:
			if r >= 128 {
				fmt.Fprintf(&buf, "\\%03o", r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte(')')
	return buf.String()
}

func HexString(data []byte) string {
	return "<" + hex.EncodeToString(data) + ">"
}

func DateString(t time.Time) string {
	_, offset := t.Zone()
	sign := '+'
	if offset < 0 {
		sign = '-'
		offset = -offset
	}
	return fmt.Sprintf("D:%s%c%02d'%02d'",
		t.Format("20060102150405"),
		sign,
		offset/3600,      //nolint:mnd
		(offset%3600)/60, //nolint:mnd
	)
}

type Stream struct {
	Dict map[string]interface{}
	Data []byte
}

var _ io.Writer = (*Encoder)(nil)
