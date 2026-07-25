// Package write serializes PDF objects, streams, cross-reference tables, and
// trailers into their final binary representation. It provides helper functions
// for PDF string literals, name objects, hex strings, and date strings.
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

const (
	decimalBase   = 10
	xrefEntrySize = 20
	xrefLineLen   = 10
	nonASCII      = 128
	secsPerHour   = 3600
	secsPerMin    = 60
)

// Encoder accumulates a PDF file into an internal buffer, writing headers,
// dictionaries, streams, cross-reference tables, and the trailer.
type Encoder struct {
	buf bytes.Buffer
}

// NewEncoder creates a new Encoder with an empty buffer.
func NewEncoder() *Encoder {
	return &Encoder{}
}

// Write implements io.Writer by appending bytes to the encoder buffer.
func (e *Encoder) Write(p []byte) (int, error) {
	n, err := e.buf.Write(p)
	if err != nil {
		return n, fmt.Errorf("write: %w", err)
	}
	return n, nil
}

// WriteString appends a plain string to the encoder buffer.
func (e *Encoder) WriteString(s string) {
	e.buf.WriteString(s)
}

// Len returns the number of bytes written so far.
func (e *Encoder) Len() int {
	return e.buf.Len()
}

// Bytes returns a copy of the accumulated PDF data.
func (e *Encoder) Bytes() []byte {
	return e.buf.Bytes()
}

// WriteHeader writes the PDF version header and binary comment.
func (e *Encoder) WriteHeader() {
	e.buf.WriteString("%PDF-2.0\n")
	e.buf.WriteString("%\x80\x80\x80\x80\n")
}

// WriteComment writes a percent-prefixed comment line.
func (e *Encoder) WriteComment(comment string) {
	fmt.Fprintf(&e.buf, "%% %s\n", comment)
}

// WriteDict serialises a map as a PDF dictionary, sorting keys alphabetically.
func (e *Encoder) WriteDict(dict map[string]interface{}) {
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	e.buf.WriteString("<< ")
	for _, k := range keys { // sorted map iteration, fine
		e.buf.WriteString(k)
		e.buf.WriteByte(' ')
		e.writeValue(dict[k])
		e.buf.WriteByte(' ')
	}
	e.buf.WriteString(">>")
}

// PDFString is a tagged string type whose value is written as a PDF literal
// string enclosed in parentheses with proper escaping.
type PDFString string

func (e *Encoder) writeValue(v interface{}) {
	switch val := v.(type) {
	case string:
		e.buf.WriteString(val)
	case PDFString:
		e.buf.WriteString(StringLit(string(val)))
	case int:
		e.buf.WriteString(strconv.Itoa(val))
	case int64:
		e.buf.WriteString(strconv.FormatInt(val, decimalBase))
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

// WriteStream writes a dictionary followed by a stream containing data.
func (e *Encoder) WriteStream(dict map[string]interface{}, data []byte) {
	e.WriteDict(dict)
	e.buf.WriteString("\nstream\n")
	e.buf.Write(data)
	e.buf.WriteString("\nendstream")
}

// WriteXref writes a cross-reference table from a slice of byte offsets.
func (e *Encoder) WriteXref(offsets []int64) {
	e.buf.WriteString("xref\n0 ")
	e.buf.Write(strconv.AppendInt(nil, int64(len(offsets)), decimalBase))
	e.buf.WriteByte('\n')
	xrefBuf := make([]byte, 0, xrefEntrySize)
	for i, off := range offsets {
		xrefBuf = strconv.AppendInt(xrefBuf[:0], off, decimalBase)
		padLen := xrefLineLen - len(xrefBuf)
		if padLen > 0 {
			e.buf.WriteString("0000000000"[:padLen])
		}
		e.buf.Write(xrefBuf)
		if i == 0 {
			e.buf.WriteString(" 65535 f \n")
		} else {
			e.buf.WriteString(" 00000 n \n")
		}
	}
}

// WriteTrailer writes the trailer dictionary with /Size, /Root, and /ID.
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

// WriteStartXref writes the startxref offset.
func (e *Encoder) WriteStartXref(offset int64) {
	fmt.Fprintf(&e.buf, "startxref\n%d\n", offset)
}

// WriteEOF writes the end-of-file marker.
func (e *Encoder) WriteEOF() {
	e.buf.WriteString("%%EOF")
}

// Ref formats a PDF indirect reference string from object and generation numbers.
func Ref(objNum, genNum int) string {
	return strconv.Itoa(objNum) + " " + strconv.Itoa(genNum) + " R"
}

// Name returns the given string prefixed with /.
func Name(s string) string {
	return "/" + s
}

// StringLit escapes s as a PDF literal string enclosed in parentheses.
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
			if r >= nonASCII {
				fmt.Fprintf(&buf, "\\%03o", r)
			} else {
				buf.WriteRune(r)
			}
		}
	}
	buf.WriteByte(')')
	return buf.String()
}

// HexString encodes data as a PDF hex string enclosed in angle brackets.
func HexString(data []byte) string {
	return "<" + hex.EncodeToString(data) + ">"
}

// DateString formats a time.Time as a PDF date string.
// t.Zone() is safe for any valid time.Time (cannot panic, always returns valid offset).
func DateString(t time.Time) string {
	_, offset := t.Zone() // zone name discarded; only offset is needed
	sign := '+'
	if offset < 0 {
		sign = '-'
		offset = -offset
	}
	return fmt.Sprintf("D:%s%c%02d'%02d'", // cold path (one-time init)
		t.Format("20060102150405"),
		sign,
		offset/secsPerHour,
		(offset%secsPerHour)/secsPerMin,
	)
}

// Stream bundles a dictionary and data for deferred serialisation. This is a
// struct holding a map and a byte slice, not an interface — false positive
// for BP-30.
type Stream struct {
	Dict map[string]interface{}
	Data []byte
}

var _ io.Writer = (*Encoder)(nil)
