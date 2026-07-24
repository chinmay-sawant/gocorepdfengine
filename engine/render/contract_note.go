// Package render maps model.ContractNote → layout tables with theme colors → multi-page PDF.
package render

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chinmay/gocorepdfengine/engine"
	"github.com/chinmay/gocorepdfengine/engine/color"
	"github.com/chinmay/gocorepdfengine/engine/doc"
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
	font    = "Helvetica"
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
		placeWatermark(start, note.Watermark)
	}

	builders, err := tl.LayOut(marginL, marginT, pageW, pageH-marginB, start)
	if err != nil {
		return nil, err
	}

	pages := make([]engine.PageContent, 0, len(builders))
	for _, b := range builders {
		pages = append(pages, engine.PageContent{
			Stream:    b.Bytes(),
			FontRes:   b.FontRes,
			UsedFonts: b.UsedFonts,
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

	return engine.GenerateDocument(engine.DocumentConfig{
		Width:    pageW,
		Height:   pageH,
		Mode:     mode,
		Title:    title,
		Author:   author,
		Subject:  subject,
		Creator:  "gocorepdfengine",
		Lang:     "en-US",
		Pages:    pages,
		UsedText: used.String(),
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

func placeWatermark(cb *layout.ContentBuilder, text string) {
	cb.PlaceText(layout.TextRun{
		Text:     text,
		FontName: font,
		FontSize: 28,
		Color:    [3]float64{0.85, 0.85, 0.85},
		X:        pageW/2 - 80,
		Y:        pageH / 2,
	})
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
	for i := 1; i < n; i++ {
		cells[i] = layout.StyledCell("", font, 10, fg, &bg, 0, h)
	}
	return layout.Row{Height: h, Cells: cells}
}

func buildRetail(note *model.ContractNote) *layout.TableLayout {
	cols := []float64{2, 1.5, 1, 1, 1.5, 1.5}
	tl := &layout.TableLayout{ColWidths: cols}

	bgH := color.ThemeHeaderBG
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 36,
		Cells: []layout.Cell{
			layout.StyledCell("CONTRACT NOTE", font, 14, color.ThemeHeaderFG, &bgH, 0, 36),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
			layout.StyledCell("CN2024001", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
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
	for i, t := range note.Trades {
		var bg *color.RGB
		if i%2 == 1 {
			c := color.ThemeAltRow
			bg = &c
		}
		afg := color.ThemeBuy
		if t.Action == "SELL" {
			afg = color.ThemeSell
		}
		tl.Rows = append(tl.Rows, layout.Row{
			Height: 14,
			Cells: []layout.Cell{
				layout.StyledCell(t.Symbol, font, 9, color.ThemeBlack, bg, 0, 14),
				layout.StyledCell(t.ISIN, font, 8, color.ThemeBlack, bg, 0, 14),
				layout.StyledCell(t.Action, font, 9, afg, bg, 0, 14),
				layout.StyledCell(strconv.Itoa(t.Qty), font, 9, color.ThemeBlack, bg, 0, 14),
				layout.StyledCell(model.Money(t.Price), font, 9, color.ThemeBlack, bg, 0, 14),
				layout.StyledCell(model.Money(t.Total), font, 9, color.ThemeBlack, bg, 0, 14),
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
	cols := []float64{2.5, 1, 1, 1.5, 1.5}
	tl := &layout.TableLayout{ColWidths: cols}
	bgH := color.ThemeHeaderBG
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 36,
		Cells: []layout.Cell{
			layout.StyledCell("ACTIVE TRADER CONTRACT NOTE", font, 12, color.ThemeHeaderFG, &bgH, 0, 36),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
			layout.StyledCell("", font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
			layout.StyledCell(fmt.Sprintf("%d Trades", len(note.Trades)), font, 10, color.ThemeHeaderSub, &bgH, 0, 36),
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
	tl.Rows = append(tl.Rows, spanProps(fmt.Sprintf("SECTION B: TRADE DETAILS (%d)", len(note.Trades)), 5, sec, color.ThemeSectionFG, 18))
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
	for i, t := range note.Trades {
		var bg *color.RGB
		if i%2 == 1 {
			c := color.ThemeAltRow
			bg = &c
		}
		afg := color.ThemeBuy
		if t.Action == "SELL" {
			afg = color.ThemeSell
		}
		tl.Rows = append(tl.Rows, layout.Row{
			Height: 12,
			Cells: []layout.Cell{
				layout.StyledCell(t.Symbol, font, 8, color.ThemeBlack, bg, 0, 12),
				layout.StyledCell(t.Action, font, 8, afg, bg, 0, 12),
				layout.StyledCell(strconv.Itoa(t.Qty), font, 8, color.ThemeBlack, bg, 0, 12),
				layout.StyledCell(model.Money(t.Price), font, 8, color.ThemeBlack, bg, 0, 12),
				layout.StyledCell(model.Money(t.Total), font, 8, color.ThemeBlack, bg, 0, 12),
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
	cols := []float64{0.6, 1, 2, 0.8, 0.6, 1.5, 1.5}
	tl := &layout.TableLayout{ColWidths: cols}
	bgH := color.ThemeHeaderBG
	tl.Rows = append(tl.Rows, layout.Row{
		Height: 32,
		Cells: []layout.Cell{
			layout.StyledCell("HFT CONTRACT NOTE", font, 11, color.ThemeHeaderFG, &bgH, 0, 32),
			layout.StyledCell("", font, 9, color.ThemeHeaderSub, &bgH, 0, 32),
			layout.StyledCell("", font, 9, color.ThemeHeaderSub, &bgH, 0, 32),
			layout.StyledCell("", font, 9, color.ThemeHeaderSub, &bgH, 0, 32),
			layout.StyledCell("", font, 9, color.ThemeHeaderSub, &bgH, 0, 32),
			layout.StyledCell(note.Client.Name, font, 9, color.ThemeHeaderSub, &bgH, 0, 32),
			layout.StyledCell(fmt.Sprintf("%d Trades", len(note.Trades)), font, 9, color.ThemeHeaderSub, &bgH, 0, 32),
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
	tl.Rows = append(tl.Rows, spanProps(fmt.Sprintf("SECTION B: TRADES (%d)", len(note.Trades)), 7, sec, color.ThemeSectionFG, 16))
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
	for i, t := range note.Trades {
		var bg *color.RGB
		if i%2 == 1 {
			c := color.ThemeAltRow
			bg = &c
		}
		afg := color.ThemeBuy
		if t.Action == "SELL" {
			afg = color.ThemeSell
		}
		tl.Rows = append(tl.Rows, layout.Row{
			Height: 10,
			Cells: []layout.Cell{
				layout.StyledCell(strconv.Itoa(t.ID), font, 7, color.ThemeBlack, bg, 0, 10),
				layout.StyledCell(t.Time, font, 7, color.ThemeBlack, bg, 0, 10),
				layout.StyledCell(t.Symbol, font, 7, color.ThemeBlack, bg, 0, 10),
				layout.StyledCell(t.Action, font, 7, afg, bg, 0, 10),
				layout.StyledCell(strconv.Itoa(t.Qty), font, 7, color.ThemeBlack, bg, 0, 10),
				layout.StyledCell(model.Money(t.Price), font, 7, color.ThemeBlack, bg, 0, 10),
				layout.StyledCell(model.Money(t.Total), font, 7, color.ThemeBlack, bg, 0, 10),
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
