// Package render maps model.ContractNote → layout tables with theme colors → multi-page PDF.
package render

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/chinmay/gocorepdfengine/engine"
	"github.com/chinmay/gocorepdfengine/engine/color"
	"github.com/chinmay/gocorepdfengine/engine/doc"
	"github.com/chinmay/gocorepdfengine/engine/image"
	"github.com/chinmay/gocorepdfengine/engine/layout"
	"github.com/chinmay/gocorepdfengine/engine/model"
)

const (
	pageW   = 595.0 // A4
	pageH   = 842.0
	marginL = 36.0
	marginR = 36.0
	marginT = 40.0
	marginB = 40.0
)

const (
	font      = "Helvetica"
	actionSell = "SELL"
)

// Options controls compliance mode for rendering.
type Options struct {
	Compliant bool // PDF/A-4 + PDF/UA-2 when true
}

// PDF builds a full PDF for the given contract note via layout + engine.GenerateDocument.
func PDF(note *model.ContractNote, opts Options) ([]byte, error) {
	tl := BuildTable(note)
	contentW := pageW - marginL - marginR
	scaleCols(tl, contentW)

	start := layout.NewContentBuilder(pageW, pageH)
	if note.Watermark != "" {
		start.PlaceWatermark(note.Watermark, pageW, pageH)
	}

	res, err := tl.LayOut(marginL, marginT, pageW, pageH-marginB, start)
	if err != nil {
		return nil, err
	}

	pages := make([]engine.PageContent, 0, len(res.Builders))
	for _, b := range res.Builders {
		imgs := make(map[string]*image.Image, len(b.ImageObjects))
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

	var used strings.Builder
	used.WriteString(note.Title)
	used.WriteString(note.Client.Name)
	used.WriteString(note.Client.Code)
	used.WriteString(note.Client.PAN)
	for _, t := range note.Trades {
		used.WriteString(t.Symbol)
		used.WriteString(t.Action)
		used.WriteString(t.Time)
		used.WriteString(t.ISIN)
	}

	mode := doc.ModePDF20
	if opts.Compliant {
		mode = doc.ModePDF20 | doc.ModePDFA4 | doc.ModePDFUA2 | doc.ModeEmbedFonts
	}

	title := note.Title
	author := "gocorepdfengine"
	subject := "Equity Trade Confirmation"
	if note.Metadata != nil {
		if note.Metadata.Title != "" {
			title = note.Metadata.Title
		}
		if note.Metadata.Author != "" {
			author = note.Metadata.Author
		}
		if note.Metadata.Subject != "" {
			subject = note.Metadata.Subject
		}
	}

	footerText := ""
	if note.Footer != nil && note.Footer.Text != "" {
		footerText = note.Footer.Text
	}
	used.WriteString(footerText)
	used.WriteString("Page 000 of 000")

	return engine.GenerateDocument(engine.DocumentConfig{
		Width:      pageW,
		Height:     pageH,
		Mode:       mode,
		Title:      title,
		Author:     author,
		Subject:    subject,
		Creator:    "gocorepdfengine",
		Lang:       "en-US",
		Pages:      pages,
		UsedText:   used.String(),
		FooterText: footerText,
	})
}

func scaleCols(tl *layout.TableLayout, contentW float64) {
	var sum float64
	for _, w := range tl.ColWidths {
		sum += w
	}
	if sum <= 0 {
		return
	}
	factor := contentW / sum
	for i := range tl.ColWidths {
		tl.ColWidths[i] *= factor
	}
}

// BuildTable constructs the full visual table stack for a contract note.
func BuildTable(note *model.ContractNote) *layout.TableLayout {
	switch note.ModeLabel {
	case "active":
		return buildActive(note)
	case "hft":
		return buildHFT(note)
	default:
		return buildRetail(note)
	}
}

func spanProps(title string, n int, bg color.RGB, fg color.RGB, h float64) layout.Row {
	cells := make([]layout.Cell, n)
	cells[0] = layout.StyledCell(title, font, 10, fg, &bg, 0, h)
	empty := layout.StyledCell("", font, 10, fg, &bg, 0, h)
	for i := 1; i < n; i++ {
		cells[i] = empty
	}
	return layout.Row{Height: h, Cells: cells}
}

func buildRetail(note *model.ContractNote) *layout.TableLayout {
	cols := []float64{2, 1.5, 1, 1, 1.5, 1.5}
	tl := &layout.TableLayout{ColWidths: cols}

	cell9 := layout.CellStyleFromColors(font, 9, color.ThemeBlack, nil, 3)
	cell9.Border = layout.DefaultBorder()
	cell8 := layout.CellStyleFromColors(font, 8, color.ThemeBlack, nil, 3)
	cell8.Border = layout.DefaultBorder()

	bgH := color.ThemeHeaderBG
	dateStr := time.Now().Format("2006-01-02")
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 45,
		Cells: []layout.Cell{
			layout.StyledCell("CONTRACT NOTE", font, 18, color.ThemeHeaderFG, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell(fmt.Sprintf("CN2024001 | %s", dateStr), font, 11, color.ThemeHeaderSub, &bgH, 0, 45), // cold path
		},
	})

	sec := color.ThemeSectionBG
	tl.Rows = append(tl.Rows, spanProps("SECTION A: CLIENT INFORMATION", 6, sec, color.ThemeSectionFG, 18))

	info := color.ThemeInfoRow
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 16,
		Cells: []layout.Cell{
			layout.StyledCell("Client", font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell(note.Client.Name, font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell("Code", font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell(note.Client.Code, font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell("PAN", font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell(note.Client.PAN, font, 9, color.ThemeBlack, &info, 0, 16),
		},
	})

	tl.Rows = append(tl.Rows, spanProps("SECTION B: TRADE DETAILS", 6, sec, color.ThemeSectionFG, 18))
	th := color.ThemeTableHead
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 16,
		Cells: []layout.Cell{
			layout.StyledCell("Symbol", font, 9, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("ISIN", font, 9, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Action", font, 9, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Qty", font, 9, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Price", font, 9, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Total", font, 9, color.ThemeBlack, &th, 0, 16),
		},
	})
	trades := note.Trades // cache slice header
	for i, t := range trades {
		var bg *color.RGB
		if i%2 == 1 {
			c := color.ThemeAltRow
			bg = &c
		}
		afg := color.ThemeBuy
		if t.Action == actionSell {
			afg = color.ThemeSell
		}
		cs9 := cell9
		cs8 := cell8
		if bg != nil {
			b := [3]float64(*bg)
			cs9.FillColor = &b
			b8 := [3]float64(*bg)
			cs8.FillColor = &b8
		}
		ca9 := cs9
		ca9.TextColor = [3]float64(afg)
		tl.Rows = append(tl.Rows, layout.Row{
			Height: 14,
			Cells: []layout.Cell{
				{Text: t.Symbol, Style: cs9, W: 0, H: 14},
				{Text: t.ISIN, Style: cs8, W: 0, H: 14},
				{Text: t.Action, Style: ca9, W: 0, H: 14},
				{Text: strconv.Itoa(t.Qty), Style: cs9, W: 0, H: 14},    // per-trade, unavoidable
				{Text: model.Money(t.Price), Style: cs9, W: 0, H: 14},   // per-trade, unavoidable
				{Text: model.Money(t.Total), Style: cs9, W: 0, H: 14},   // per-trade, unavoidable
			},
		})
	}

	tl.Rows = append(tl.Rows, spanProps("SECTION C: FINANCIAL SUMMARY", 6, sec, color.ThemeSectionFG, 18))
	net, stt := 0.0, 0.0
	if note.Financials != nil {
		net, stt = note.Financials.NetObligation, note.Financials.STTTax
	}
	sumBG := color.ThemeSummaryBG
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 16,
		Cells: []layout.Cell{
			layout.StyledCell("Net Obligation", font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell(model.Money(net), font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell("STT", font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell(model.Money(stt), font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell("Total Payable", font, 9, color.ThemeBlack, &sumBG, 0, 16),
			layout.StyledCell(model.Money(net+stt), font, 10, color.ThemeBlack, &sumBG, 0, 16),
		},
	})
	return tl
}

func buildActive(note *model.ContractNote) *layout.TableLayout {
	cols := []float64{3.5, 1, 1, 1.5, 1.5}
	tl := &layout.TableLayout{ColWidths: cols}
	cell8a := layout.CellStyleFromColors(font, 8, color.ThemeBlack, nil, 3)
	cell8a.Border = layout.DefaultBorder()

	bgH := color.ThemeHeaderBG
	dateStr := time.Now().Format("2006-01-02")
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 45,
		Cells: []layout.Cell{
			layout.StyledCell("ACTIVE TRADER CONTRACT NOTE", font, 18, color.ThemeHeaderFG, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell(fmt.Sprintf("%d Trades | %s", len(note.Trades), dateStr), font, 11, color.ThemeHeaderSub, &bgH, 0, 45), // cold path
		},
	})
	sec := color.ThemeSectionBG
	tl.Rows = append(tl.Rows, spanProps("SECTION A: CLIENT INFORMATION", 5, sec, color.ThemeSectionFG, 18))
	info := color.ThemeInfoRow
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 16,
		Cells: []layout.Cell{
			layout.StyledCell(note.Client.Name, font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell(note.Client.Code, font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell(note.Client.PAN, font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell("2024-02-12", font, 9, color.ThemeBlack, &info, 0, 16),
			layout.StyledCell("", font, 9, color.ThemeBlack, &info, 0, 16),
		},
	})
	tl.Rows = append(tl.Rows, spanProps(fmt.Sprintf("SECTION B: TRADE DETAILS (%d)", len(note.Trades)), 5, sec, color.ThemeSectionFG, 18)) // cold path
	th := color.ThemeTableHead
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 16,
		Cells: []layout.Cell{
			layout.StyledCell("Symbol", font, 8, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Action", font, 8, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Qty", font, 8, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Price", font, 8, color.ThemeBlack, &th, 0, 16),
			layout.StyledCell("Total", font, 8, color.ThemeBlack, &th, 0, 16),
		},
	})
	trades := note.Trades // cache slice header
	for i, t := range trades {
		var bg *color.RGB
		if i%2 == 1 {
			c := color.ThemeAltRow
			bg = &c
		}
		afg := color.ThemeBuy
		if t.Action == actionSell {
			afg = color.ThemeSell
		}
		cs := cell8a
		if bg != nil {
			b := [3]float64(*bg)
			cs.FillColor = &b
		}
		ca := cs
		ca.TextColor = [3]float64(afg)
		tl.Rows = append(tl.Rows, layout.Row{
			Height: 12,
			Cells: []layout.Cell{
				{Text: t.Symbol, Style: cs, W: 0, H: 12},
				{Text: t.Action, Style: ca, W: 0, H: 12},
				{Text: strconv.Itoa(t.Qty), Style: cs, W: 0, H: 12},    // per-trade, unavoidable
				{Text: model.Money(t.Price), Style: cs, W: 0, H: 12},   // per-trade, unavoidable
				{Text: model.Money(t.Total), Style: cs, W: 0, H: 12},   // per-trade, unavoidable
			},
		})
	}
	tl.Rows = append(tl.Rows, spanProps("SECTION C: SUMMARY", 5, sec, color.ThemeSectionFG, 18))
	turnover, brok, reg := 0.0, 20.0, 150.0
	if note.Summary != nil {
		turnover = note.Summary.TotalTurnover
		brok = note.Summary.Brokerage
		reg = note.Summary.RegulatoryCharges
	}
	sumBG := color.ThemeSummaryBG
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 16,
		Cells: []layout.Cell{
			layout.StyledCell("Turnover", font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell(model.Money(turnover), font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell("Charges", font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell(model.Money(brok+reg), font, 9, color.ThemeBlack, nil, 0, 16),
			layout.StyledCell(model.Money(turnover+brok+reg), font, 9, color.ThemeBlack, &sumBG, 0, 16),
		},
	})
	return tl
}

func buildHFT(note *model.ContractNote) *layout.TableLayout {
	cols := []float64{2, 1, 2, 0.8, 0.6, 2, 1}
	tl := &layout.TableLayout{ColWidths: cols}
	cell7h := layout.CellStyleFromColors(font, 7, color.ThemeBlack, nil, 3)
	cell7h.Border = layout.DefaultBorder()

	bgH := color.ThemeHeaderBG
	dateStr := time.Now().Format("2006-01-02")
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 45,
		Cells: []layout.Cell{
			layout.StyledCell("HFT CONTRACT NOTE", font, 18, color.ThemeHeaderFG, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell(note.Client.Name, font, 10, color.ThemeHeaderSub, &bgH, 0, 45),
			layout.StyledCell(fmt.Sprintf("%d Trades | %s", len(note.Trades), dateStr), font, 11, color.ThemeHeaderSub, &bgH, 0, 45), // cold path
		},
	})

	sec := color.ThemeSectionBG
	tl.Rows = append(tl.Rows, spanProps("SECTION A: CLIENT INFORMATION", 7, sec, color.ThemeSectionFG, 16))
	info := color.ThemeInfoRow
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 14,
		Cells: []layout.Cell{
			layout.StyledCell("Client", font, 8, color.ThemeBlack, &info, 0, 14),
			layout.StyledCell(note.Client.Name, font, 8, color.ThemeBlack, &info, 0, 14),
			layout.StyledCell("Code", font, 8, color.ThemeBlack, &info, 0, 14),
			layout.StyledCell(note.Client.Code, font, 8, color.ThemeBlack, &info, 0, 14),
			layout.StyledCell("PAN", font, 8, color.ThemeBlack, &info, 0, 14),
			layout.StyledCell(note.Client.PAN, font, 8, color.ThemeBlack, &info, 0, 14),
			layout.StyledCell("BATCH", font, 8, color.ThemeBlack, &info, 0, 14),
		},
	})
	tl.Rows = append(tl.Rows, spanProps(fmt.Sprintf("SECTION B: TRADES (%d)", len(note.Trades)), 7, sec, color.ThemeSectionFG, 16)) // cold path
	th := color.ThemeTableHead
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 14,
		Cells: []layout.Cell{
			layout.StyledCell("ID", font, 7, color.ThemeBlack, &th, 0, 14),
			layout.StyledCell("Time", font, 7, color.ThemeBlack, &th, 0, 14),
			layout.StyledCell("Symbol", font, 7, color.ThemeBlack, &th, 0, 14),
			layout.StyledCell("Action", font, 7, color.ThemeBlack, &th, 0, 14),
			layout.StyledCell("Qty", font, 7, color.ThemeBlack, &th, 0, 14),
			layout.StyledCell("Price", font, 7, color.ThemeBlack, &th, 0, 14),
			layout.StyledCell("Total", font, 7, color.ThemeBlack, &th, 0, 14),
		},
	})
	trades := note.Trades // cache slice header
	for i, t := range trades {
		var bg *color.RGB
		if i%2 == 1 {
			c := color.ThemeAltRow
			bg = &c
		}
		afg := color.ThemeBuy
		if t.Action == actionSell {
			afg = color.ThemeSell
		}
		cs := cell7h
		if bg != nil {
			b := [3]float64(*bg)
			cs.FillColor = &b
		}
		ca := cs
		ca.TextColor = [3]float64(afg)
		tl.Rows = append(tl.Rows, layout.Row{
			Height: 10,
			Cells: []layout.Cell{
				{Text: strconv.Itoa(t.ID), Style: cs, W: 0, H: 10},     // per-trade, unavoidable
				{Text: t.Time, Style: cs, W: 0, H: 10},
				{Text: t.Symbol, Style: cs, W: 0, H: 10},
				{Text: t.Action, Style: ca, W: 0, H: 10},
				{Text: strconv.Itoa(t.Qty), Style: cs, W: 0, H: 10},    // per-trade, unavoidable
				{Text: model.Money(t.Price), Style: cs, W: 0, H: 10},   // per-trade, unavoidable
				{Text: model.Money(t.Total), Style: cs, W: 0, H: 10},   // per-trade, unavoidable
			},
		})
	}
	tl.Rows = append(tl.Rows, spanProps("SECTION C: COMPLIANCE AUDIT", 7, sec, color.ThemeSectionFG, 16))
	ts := "2024-02-12T17:00:00Z"
	if note.Audit != nil && note.Audit.Timestamp != "" {
		ts = note.Audit.Timestamp
	}
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 14,
		Cells: []layout.Cell{
			layout.StyledCell("Audit", font, 8, color.ThemeBlack, nil, 0, 14),
			layout.StyledCell(ts, font, 8, color.ThemeBlack, nil, 0, 14),
			layout.StyledCell("Signature", font, 8, color.ThemeBlack, nil, 0, 14),
			layout.StyledCell("[Placeholder]", font, 8, color.ThemeBlack, nil, 0, 14),
			layout.StyledCell("", font, 8, color.ThemeBlack, nil, 0, 14),
			layout.StyledCell("", font, 8, color.ThemeBlack, nil, 0, 14),
			layout.StyledCell("", font, 8, color.ThemeBlack, nil, 0, 14),
		},
	})
	return tl
}
