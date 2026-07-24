package model

import (
	"encoding/json"
	"fmt"
	"os"
)

type PageSize struct {
	Width, Height float64
}

var PageSizes = map[string]PageSize{
	"A3":     {842, 1191},
	"A4":     {595, 842},
	"A5":     {420, 595},
	"LETTER": {612, 792},
	"LEGAL":  {612, 1008},
}

type Config struct {
	PageBorder          string `json:"pageBorder"`
	Page                string `json:"page"`
	PageAlignment       int    `json:"pageAlignment"`
	Watermark           string `json:"watermark"`
	PDFTitle            string `json:"pdfTitle"`
	PDFACompliant       bool   `json:"pdfaCompliant"`
	ArlingtonCompatible bool   `json:"arlingtonCompatible"`
	EmbedFonts          bool   `json:"embedFonts"`
}

// PageSize returns the page width and height for the configured page size name
// (A4, A3, etc.), defaulting to A4 when unset or unknown.
func (c *Config) PageSize() (float64, float64) {
	if c.Page == "" {
		c.Page = "A4"
	}
	ps, ok := PageSizes[c.Page]
	if !ok {
		ps = PageSizes["A4"]
	}
	return ps.Width, ps.Height
}

type Title struct {
	Props     string           `json:"props"`
	Text      string           `json:"text"`
	Table     *TableDef        `json:"table,omitempty"`
	BGColor   string           `json:"bgcolor,omitempty"`
	TextColor string           `json:"textcolor,omitempty"`
	Link      string           `json:"link,omitempty"`
}

type TableDef struct {
	MaxColumns   int        `json:"maxcolumns"`
	ColumnWidths []float64  `json:"columnwidths"`
	RowHeights   []float64  `json:"rowheights,omitempty"`
	BGColor      string     `json:"bgcolor,omitempty"`
	TextColor    string     `json:"textcolor,omitempty"`
	Rows         []TableRow `json:"rows"`
}

type TableRow struct {
	Row []TableCell `json:"row"`
}

type TableCell struct {
	Props     string `json:"props"`
	Text      string `json:"text,omitempty"`
	BGColor   string `json:"bgcolor,omitempty"`
	TextColor string `json:"textcolor,omitempty"`
	Width     float64 `json:"width,omitempty"`
	Height    float64 `json:"height,omitempty"`
	Link      string `json:"link,omitempty"`
	Dest      string `json:"dest,omitempty"`
	Image     *Image `json:"image,omitempty"`
}

type Image struct {
	ImageName string `json:"imagename,omitempty"`
	ImageData string `json:"imagedata"`
	Width     float64 `json:"width"`
	Height    float64 `json:"height"`
}

type Spacer struct {
	Height float64 `json:"height"`
}

type Element struct {
	Type  string `json:"type"`
	Index int    `json:"index"`
}

type PDFTemplate struct {
	Config   *Config    `json:"config"`
	Title    *Title     `json:"title"`
	Tables   []TableDef `json:"table"`
	Spacers  []Spacer   `json:"spacer"`
	Elements []Element  `json:"elements"`
	Footer   *Footer    `json:"footer,omitempty"`
}

// LoadTemplate reads a JSON file at path and unmarshals it into a PDFTemplate.
func LoadTemplate(path string) (*PDFTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("template: reading %s: %w", path, err)
	}
	var t PDFTemplate
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("template: parse %s: %w", path, err)
	}
	if t.Config == nil {
		t.Config = &Config{}
	}
	return &t, nil
}
