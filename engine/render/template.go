package render

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/chinmay/gocorepdfengine/engine"
	"github.com/chinmay/gocorepdfengine/engine/color"
	"github.com/chinmay/gocorepdfengine/engine/doc"
	"github.com/chinmay/gocorepdfengine/engine/image"
	"github.com/chinmay/gocorepdfengine/engine/layout"
	"github.com/chinmay/gocorepdfengine/engine/model"
)

func TemplatePDF(t *model.PDFTemplate, opts Options) ([]byte, error) {
	pageW, pageH := pageDimensions(t.Config)
	contentW := pageW - marginL - marginR

	tables := buildTables(t, contentW)
	start := layout.NewContentBuilder(pageW, pageH)
	if t.Config.Watermark != "" {
		start.PlaceWatermark(t.Config.Watermark, pageW, pageH)
	}

	var allBuilders []*layout.ContentBuilder
	var cur = start
	y := pageH - marginT

	for _, tbl := range tables {
		if tbl == nil {
			continue
		}
		var res layout.LayoutResult
		var err error
		if len(allBuilders) == 0 {
			res, err = tbl.LayOut(marginL, marginT, pageW, pageH, cur)
		} else {
			res, err = tbl.LayOutFrom(marginL, marginT, pageW, pageH, y, cur)
		}
		if err != nil {
			return nil, fmt.Errorf("template layout: %w", err)
		}
		if len(allBuilders) == 0 {
			allBuilders = res.Builders
		} else {
			allBuilders = append(allBuilders, res.Builders[1:]...)
		}
		cur = allBuilders[len(allBuilders)-1]
		y = res.Y
	}

	pages := make([]engine.PageContent, 0, len(allBuilders))
	for _, b := range allBuilders {
		imgs := make(map[string]*image.Image)
		for name, obj := range b.ImageObjects {
			imgs[name] = obj.Img
		}
		pages = append(pages, engine.PageContent{
			Stream:        b.Bytes(),
			FontRes:       b.FontRes,
			UsedFonts:     b.UsedFonts,
			ImageXObjects: imgs,
		})
	}

	mode := doc.ModePDF20
	if t.Config.PDFACompliant || t.Config.ArlingtonCompatible {
		mode |= doc.ModePDFA4 | doc.ModePDFUA2 | doc.ModeEmbedFonts
	}

	footerText := ""
	if t.Footer != nil {
		footerText = t.Footer.Text
	}

	used := collectUsed(t)

	return engine.GenerateDocument(engine.DocumentConfig{
		Width:      pageW,
		Height:     pageH,
		Mode:       mode,
		Title:      t.Config.PDFTitle,
		Author:     "gocorepdfengine",
		Subject:    t.Config.PDFTitle,
		Creator:    "gocorepdfengine",
		Lang:       "en-US",
		Pages:      pages,
		UsedText:   used,
		FooterText: footerText,
	})
}

func pageDimensions(cfg *model.Config) (float64, float64) {
	return cfg.PageSize()
}

func buildTables(t *model.PDFTemplate, contentW float64) []*layout.TableLayout {
	var tables []*layout.TableLayout

	if t.Title != nil {
		tables = append(tables, titleLayout(t.Title, contentW))
	}

	for _, elem := range t.Elements {
		switch elem.Type {
		case "table":
			if elem.Index >= 0 && elem.Index < len(t.Tables) {
				tables = append(tables, tableLayout(&t.Tables[elem.Index], contentW))
			}
		case "spacer":
			if elem.Index >= 0 && elem.Index < len(t.Spacers) {
				tables = append(tables, spacerLayout(t.Spacers[elem.Index].Height, contentW))
			}
		}
	}

	return tables
}

func titleLayout(title *model.Title, contentW float64) *layout.TableLayout {
	if title.Table != nil {
		return tableLayout(title.Table, contentW)
	}
	if title.Text == "" {
		return nil
	}

	cols := []float64{1}
	tl := &layout.TableLayout{ColWidths: cols}

	var fill *color.RGB
	if title.BGColor != "" {
		c, err := color.ParseHex(title.BGColor)
		if err == nil {
			fill = &c
		}
	}
	var tc [3]float64
	if title.TextColor != "" {
		c, err := color.ParseHex(title.TextColor)
		if err == nil {
			tc = [3]float64(c)
		}
	}

	p, _ := layout.ParseProps(title.Props)
	rowH := p.FontSize*2 + 12
	if rowH < 36 {
		rowH = 36
	}

	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowH,
		Cells: []layout.Cell{
			cellFromProps(title.Text, p, &tc, fill, contentW, rowH),
		},
	})
	return tl
}

func tableLayout(td *model.TableDef, contentW float64) *layout.TableLayout {
	// Determine column widths: use explicit cell widths if present,
	// otherwise scale relative weights to content width.
	weights := resolveColWidths(td, contentW)
	tl := &layout.TableLayout{ColWidths: weights}

	var defaultBG *color.RGB
	if td.BGColor != "" {
		c, err := color.ParseHex(td.BGColor)
		if err == nil {
			defaultBG = &c
		}
	}
	var defaultTC [3]float64
	if td.TextColor != "" {
		c, err := color.ParseHex(td.TextColor)
		if err == nil {
			defaultTC = [3]float64(c)
		}
	}

	for i, row := range td.Rows {
		rowH := 0.0
		if i < len(td.RowHeights) && td.RowHeights[i] > 0 {
			rowH = td.RowHeights[i]
		}
		for _, c := range row.Row {
			if c.Height > rowH {
				rowH = c.Height
			}
		}
		if rowH <= 0 {
			rowH = 25
		}

		r := layout.Row{Height: rowH}
		for _, c := range row.Row {
			p, _ := layout.ParseProps(c.Props)
			var fill *color.RGB
			if c.BGColor != "" {
				parsed, err := color.ParseHex(c.BGColor)
				if err == nil {
					fill = &parsed
				}
			} else if defaultBG != nil {
				fill = defaultBG
			}
			var tc [3]float64
			if c.TextColor != "" {
				parsed, err := color.ParseHex(c.TextColor)
				if err == nil {
					tc = [3]float64(parsed)
				}
			} else if defaultTC != [3]float64{} {
				tc = defaultTC
			}
			lc := cellFromProps(c.Text, p, &tc, fill, c.Width, rowH)
			if c.Image != nil && c.Image.ImageData != "" {
				raw, err := base64.StdEncoding.DecodeString(c.Image.ImageData)
				if err == nil && len(raw) > 0 {
					isJPEG := len(raw) > 2 && raw[0] == 0xFF && raw[1] == 0xD8
					lc.Image = &layout.CellImage{Data: raw, IsJPEG: isJPEG}
				}
			}
			r.Cells = append(r.Cells, lc)
		}

		for len(r.Cells) < td.MaxColumns {
			r.Cells = append(r.Cells, layout.Cell{Style: layout.CellStyle{}})
		}
		tl.Rows = append(tl.Rows, r)
	}

	return tl
}

func spacerLayout(height, contentW float64) *layout.TableLayout {
	return &layout.TableLayout{
		ColWidths: []float64{contentW},
		Rows: []layout.Row{
			{Height: height, Cells: []layout.Cell{{Style: layout.CellStyle{}}}},
		},
	}
}

// resolveColWidths returns base column widths from table-level ColumnWidths.
// Individual cell Width overrides are handled per-cell in the layout engine.
func resolveColWidths(td *model.TableDef, contentW float64) []float64 {
	weights := td.ColumnWidths
	if len(weights) == 0 {
		weights = make([]float64, td.MaxColumns)
		for i := range weights {
			weights[i] = 1
		}
	}
	scaleColsWeights(weights, contentW)
	return weights
}

func scaleColsWeights(weights []float64, total float64) {
	var sum float64
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		return
	}
	factor := total / sum
	for i := range weights {
		weights[i] *= factor
	}
}

func cellFromProps(text string, p layout.CellProps, tc *[3]float64, fill *color.RGB, width, rowH float64) layout.Cell {
	style := layout.CellStyle{
		FontName: p.FontName,
		FontSize: p.FontSize,
		Align:    p.Align,
	}
	if tc != nil {
		style.TextColor = *tc
	}
	if fill != nil {
		f := [3]float64(*fill)
		style.FillColor = &f
	}
	bw := 0.5
	bc := [3]float64{0.3, 0.3, 0.3}
	// Props order (positions 5-8): left, right, top, bottom
	if p.Border[0] {
		style.BorderLeft = &layout.BorderStyle{Width: bw, Color: bc}
	}
	if p.Border[1] {
		style.BorderRight = &layout.BorderStyle{Width: bw, Color: bc}
	}
	if p.Border[2] {
		style.BorderTop = &layout.BorderStyle{Width: bw, Color: bc}
	}
	if p.Border[3] {
		style.BorderBottom = &layout.BorderStyle{Width: bw, Color: bc}
	}
	return layout.Cell{Text: text, Style: style, W: width, H: rowH}
}

func collectUsed(t *model.PDFTemplate) string {
	var b strings.Builder
	if t.Title != nil {
		b.WriteString(t.Title.Text)
	}
	for _, td := range t.Tables {
		for _, row := range td.Rows {
			for _, c := range row.Row {
				b.WriteString(c.Text)
			}
		}
	}
	if t.Footer != nil {
		b.WriteString(t.Footer.Text)
	}
	b.WriteString("Page 000 of 000")
	return b.String()
}
