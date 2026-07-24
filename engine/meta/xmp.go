package meta

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"strings"
	"text/template"
	"time"
)

type XMPConfig struct {
	Title      string
	Author     string
	Subject    string
	Keywords   string
	Creator    string
	Producer   string
	CreateDate time.Time
	ModifyDate time.Time
	PDFA       bool
	PDFUA      bool
	DocumentID string
	InstanceID string
}

// DefaultConfig returns an XMPConfig populated with sensible defaults:
// current timestamps, random UUIDs, and the engine name as creator/producer.
func DefaultConfig() XMPConfig {
	now := time.Now()
	return XMPConfig{
		Creator:    "gocorepdfengine",
		Producer:   "gocorepdfengine",
		CreateDate: now,
		ModifyDate: now,
		DocumentID: "uuid:" + newUUID(),
		InstanceID: "uuid:" + newUUID(),
	}
}

const xmpTemplateStr = "\xef\xbb\xbf<?xpacket begin=\"\xef\xbb\xbf\" id=\"W5M0MpCehiHzreSzNTczkc9d\"?>\n<x:xmpmeta xmlns:x=\"adobe:ns:meta/\">\n  <rdf:RDF xmlns:rdf=\"http://www.w3.org/1999/02/22-rdf-syntax-ns#\">\n    <rdf:Description rdf:about=\"\"\n      xmlns:xmp=\"http://ns.adobe.com/xap/1.0/\"\n      xmlns:dc=\"http://purl.org/dc/elements/1.1/\"\n      xmlns:pdf=\"http://ns.adobe.com/pdf/1.3/\"\n      xmlns:xmpMM=\"http://ns.adobe.com/xap/1.0/mm/\"{{if .PDFA}}\n      xmlns:pdfaid=\"http://www.aiim.org/pdfa/ns/id/\"{{end}}{{if .PDFUA}}\n      xmlns:pdfuaid=\"http://www.aiim.org/pdfua/ns/id/\"\n      xmlns:pdfaExtension=\"http://www.aiim.org/pdfa/ns/extension/\"{{end}}>\n      <xmp:CreateDate>{{.CreateDate}}</xmp:CreateDate>\n      <xmp:ModifyDate>{{.ModifyDate}}</xmp:ModifyDate>\n      <xmp:MetadataDate>{{.MetadataDate}}</xmp:MetadataDate>\n      <xmp:CreatorTool>{{escape .Creator}}</xmp:CreatorTool>\n      <dc:format>application/pdf</dc:format>\n{{if .Title}}      <dc:title>\n        <rdf:Alt>\n          <rdf:li xml:lang=\"x-default\">{{escape .Title}}</rdf:li>\n        </rdf:Alt>\n      </dc:title>\n{{end}}{{if .Author}}      <dc:creator>\n        <rdf:Seq>\n          <rdf:li>{{escape .Author}}</rdf:li>\n        </rdf:Seq>\n      </dc:creator>\n{{end}}{{if .Subject}}      <dc:description>\n        <rdf:Alt>\n          <rdf:li xml:lang=\"x-default\">{{escape .Subject}}</rdf:li>\n        </rdf:Alt>\n      </dc:description>\n{{end}}{{if .Keywords}}      <pdf:Keywords>{{escape .Keywords}}</pdf:Keywords>\n{{end}}      <pdf:Producer>{{escape .Producer}}</pdf:Producer>\n      <xmpMM:DocumentID>{{escape .DocumentID}}</xmpMM:DocumentID>\n      <xmpMM:InstanceID>{{escape .InstanceID}}</xmpMM:InstanceID>\n{{if .PDFA}}      <pdfaid:part>4</pdfaid:part>\n      <pdfaid:rev>2020</pdfaid:rev>\n{{end}}{{if .PDFUA}}      <pdfuaid:part>2</pdfuaid:part>\n      <pdfuaid:rev>2024</pdfuaid:rev>\n      <pdfaExtension:schemas>\n        <rdf:Bag>\n          <rdf:li rdf:parseType=\"Resource\">\n            <pdfaExtension:namespaceURI>http://www.aiim.org/pdfua/ns/id/</pdfaExtension:namespaceURI>\n            <pdfaExtension:prefix>pdfuaid</pdfaExtension:prefix>\n            <pdfaExtension:property>\n              <rdf:Seq>\n                <rdf:li rdf:parseType=\"Resource\">\n                  <pdfaExtension:name>part</pdfaExtension:name>\n                  <pdfaExtension:valueType>Integer</pdfaExtension:valueType>\n                  <pdfaExtension:category>internal</pdfaExtension:category>\n                  <pdfaExtension:description>Part of PDF UA standard</pdfaExtension:description>\n                </rdf:li>\n                <rdf:li rdf:parseType=\"Resource\">\n                  <pdfaExtension:name>rev</pdfaExtension:name>\n                  <pdfaExtension:valueType>Integer</pdfaExtension:valueType>\n                  <pdfaExtension:category>internal</pdfaExtension:category>\n                  <pdfaExtension:description>Revision of PDF UA standard</pdfaExtension:description>\n                </rdf:li>\n              </rdf:Seq>\n            </pdfaExtension:property>\n          </rdf:li>\n        </rdf:Bag>\n      </pdfaExtension:schemas>\n{{end}}    </rdf:Description>\n  </rdf:RDF>\n</x:xmpmeta>\n{{.Padding}}<?xpacket end=\"w\"?>"

type xmpData struct {
	Title        string
	Author       string
	Subject      string
	Keywords     string
	Creator      string
	Producer     string
	CreateDate   string
	ModifyDate   string
	MetadataDate string
	PDFA         bool
	PDFUA        bool
	DocumentID   string
	InstanceID   string
	Padding      string
}

var xmpTmpl = template.Must(template.New("xmp").Funcs(template.FuncMap{
	"escape": xmlEscape,
}).Parse(xmpTemplateStr))

func xmlEscape(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}

func fmtISO8601(t time.Time) string {
	return t.Format("2006-01-02T15:04:05-07:00")
}

func newUUID() string {
	var u [16]byte
	if _, err := rand.Read(u[:]); err != nil {
		panic("meta: crypto/rand.Read failed: " + err.Error())
	}
	u[6] = (u[6] & 0x0f) | 0x40
	u[8] = (u[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", // cold path (one-time UUID gen)
		u[0:4], u[4:6], u[6:8], u[8:10], u[10:16])
}

func padding(n int) string {
	var buf bytes.Buffer
	for buf.Len() < n {
		buf.WriteString("                                                                                \n")
	}
	return buf.String()[:n]
}

// BuildXMP renders the XMP metadata XML from the given config, including
// the xpacket wrapper and padding.
func BuildXMP(config XMPConfig) []byte {
	md := config.ModifyDate
	if md.IsZero() {
		md = config.CreateDate
	}
	data := xmpData{
		Title:        config.Title,
		Author:       config.Author,
		Subject:      config.Subject,
		Keywords:     config.Keywords,
		Creator:      config.Creator,
		Producer:     config.Producer,
		CreateDate:   fmtISO8601(config.CreateDate),
		ModifyDate:   fmtISO8601(config.ModifyDate),
		MetadataDate: fmtISO8601(md),
		PDFA:         config.PDFA,
		PDFUA:        config.PDFUA,
		DocumentID:   config.DocumentID,
		InstanceID:   config.InstanceID,
		Padding:      padding(2048),
	}
	var buf bytes.Buffer
	if err := xmpTmpl.Execute(&buf, data); err != nil {
		panic("meta: xmp template execute: " + err.Error())
	}
	return buf.Bytes()
}
