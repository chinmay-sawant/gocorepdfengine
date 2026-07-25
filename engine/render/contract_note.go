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
	font       = "Helvetica"
	actionSell = "SELL"
)

const (
	fontSizeHeaderHFT    = 18
	fontSizeHeaderSub    = 11
	fontSizeSectionLabel = 7
	fontSizeTableCellHFT = 10
	fontSizeTableCell    = 8
	fontSizeTableCellSm  = 7
	fontSizeSummaryLabel = 9
	fontSizeSectionPad   = 5
	fontSizeClientInfo   = 9
)

const (
	rowHeightHeader     = 45
	rowHeightSectionHdr = 18
	rowHeightSub        = 16
	rowHeightTradeRow   = 14
	rowHeightTradeHFT   = 12
	rowHeightAudit      = 14
	rowHeightSummary    = 16
	rowHeightClient     = 16
	rowHeightTradeSm    = 10
)

const (
	propSpanRetail = 6
	propSpanActive = 5
	propSpanHFT    = 7
	colSpan3       = 3
)

const (
	decimalBase        = 10
	defaultCellPadding = 3
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
		// codehound-ignore: PERF-35
		return nil, fmt.Errorf("contract note layout: %w", err)
	}

	pages := make([]engine.PageContent, 0, len(res.Builders))
	for _, b := range res.Builders {
		imgs := make(map[string]*image.Image, len(b.ImageObjects)) // BP-52: pre-sized to known count
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

	pdf, err := engine.GenerateDocument(engine.DocumentConfig{
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
	if err != nil {
		return nil, fmt.Errorf("generate document: %w", err)
	}
	return pdf, nil
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
	cells[0] = layout.StyledCell(title, font, fontSizeTableCellHFT, fg, &bg, 0, h)
	empty := layout.StyledCell("", font, fontSizeTableCellHFT, fg, &bg, 0, h)
	for i := 1; i < n; i++ {
		cells[i] = empty
	}
	return layout.Row{Height: h, Cells: cells}
}

func buildRetail(note *model.ContractNote) *layout.TableLayout {
	cols := []float64{2, 1.5, 1, 1, 1.5, 1.5}
	tl := &layout.TableLayout{ColWidths: cols}

	cell9 := layout.CellStyleFromColors(font, fontSizeSummaryLabel, color.ThemeBlack, nil, defaultCellPadding)
	cell9.Border = layout.DefaultBorder()
	cell8 := layout.CellStyleFromColors(font, fontSizeTableCell, color.ThemeBlack, nil, defaultCellPadding)
	cell8.Border = layout.DefaultBorder()

	bgH := color.ThemeHeaderBG
	dateStr := time.Now().Format("2006-01-02")
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightHeader,
		Cells: []layout.Cell{
			layout.StyledCell("CONTRACT NOTE", font, fontSizeHeaderHFT, color.ThemeHeaderFG, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("CN2024001 | "+dateStr, font, fontSizeHeaderSub, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader), // cold path
		},
	})

	sec := color.ThemeSectionBG
	tl.Rows = append(tl.Rows, spanProps("SECTION A: CLIENT INFORMATION", propSpanRetail, sec, color.ThemeSectionFG, rowHeightSectionHdr))

	info := color.ThemeInfoRow
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightSub,
		Cells: []layout.Cell{
			layout.StyledCell("Client", font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightSub),
			layout.StyledCell(note.Client.Name, font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightSub),
			layout.StyledCell("Code", font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightSub),
			layout.StyledCell(note.Client.Code, font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightSub),
			layout.StyledCell("PAN", font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightSub),
			layout.StyledCell(note.Client.PAN, font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightSub),
		},
	})

	tl.Rows = append(tl.Rows, spanProps("SECTION B: TRADE DETAILS", propSpanRetail, sec, color.ThemeSectionFG, rowHeightSectionHdr))
	th := color.ThemeTableHead
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightSub,
		Cells: []layout.Cell{
			layout.StyledCell("Symbol", font, fontSizeSummaryLabel, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("ISIN", font, fontSizeSummaryLabel, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Action", font, fontSizeSummaryLabel, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Qty", font, fontSizeSummaryLabel, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Price", font, fontSizeSummaryLabel, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Total", font, fontSizeSummaryLabel, color.ThemeBlack, &th, 0, rowHeightSub),
		},
	})
	trades := note.Trades
	var tradeBuf []byte
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
		tradeBuf = strconv.AppendInt(tradeBuf[:0], int64(t.Qty), decimalBase)
		tl.Rows = append(tl.Rows, layout.Row{
			Height: rowHeightTradeRow,
			Cells: []layout.Cell{
				{Text: t.Symbol, Style: cs9, W: 0, H: rowHeightTradeRow},
				{Text: t.ISIN, Style: cs8, W: 0, H: rowHeightTradeRow},
				{Text: t.Action, Style: ca9, W: 0, H: rowHeightTradeRow},
				{Text: string(tradeBuf), Style: cs9, W: 0, H: rowHeightTradeRow},
				{Text: model.Money(t.Price), Style: cs9, W: 0, H: rowHeightTradeRow},
				{Text: model.Money(t.Total), Style: cs9, W: 0, H: rowHeightTradeRow},
			},
		})
	}

	tl.Rows = append(tl.Rows, spanProps("SECTION C: FINANCIAL SUMMARY", propSpanRetail, sec, color.ThemeSectionFG, rowHeightSectionHdr))
	net, stt := 0.0, 0.0
	if note.Financials != nil {
		net, stt = note.Financials.NetObligation, note.Financials.STTTax
	}
	sumBG := color.ThemeSummaryBG
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightSummary,
		Cells: []layout.Cell{
			layout.StyledCell("Net Obligation", font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell(model.Money(net), font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell("STT", font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell(model.Money(stt), font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell("Total Payable", font, fontSizeSummaryLabel, color.ThemeBlack, &sumBG, 0, rowHeightSummary),
			layout.StyledCell(model.Money(net+stt), font, fontSizeTableCellHFT, color.ThemeBlack, &sumBG, 0, rowHeightSummary),
		},
	})
	return tl
}

func buildActive(note *model.ContractNote) *layout.TableLayout {
	cols := []float64{3.5, 1, 1, 1.5, 1.5}
	tl := &layout.TableLayout{ColWidths: cols}
	cell8a := layout.CellStyleFromColors(font, fontSizeTableCell, color.ThemeBlack, nil, defaultCellPadding)
	cell8a.Border = layout.DefaultBorder()

	bgH := color.ThemeHeaderBG
	dateStr := time.Now().Format("2006-01-02")
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightHeader,
		Cells: []layout.Cell{
			layout.StyledCell("ACTIVE TRADER CONTRACT NOTE", font, fontSizeHeaderHFT, color.ThemeHeaderFG, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell(fmt.Sprintf("%d Trades | %s", len(note.Trades), dateStr), font, fontSizeHeaderSub, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader), // cold path
		},
	})
	sec := color.ThemeSectionBG
	tl.Rows = append(tl.Rows, spanProps("SECTION A: CLIENT INFORMATION", propSpanActive, sec, color.ThemeSectionFG, rowHeightSectionHdr))
	info := color.ThemeInfoRow
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightClient,
		Cells: []layout.Cell{
			layout.StyledCell(note.Client.Name, font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightClient),
			layout.StyledCell(note.Client.Code, font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightClient),
			layout.StyledCell(note.Client.PAN, font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightClient),
			layout.StyledCell("2024-02-12", font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightClient),
			layout.StyledCell("", font, fontSizeClientInfo, color.ThemeBlack, &info, 0, rowHeightClient),
		},
	})
	tl.Rows = append(tl.Rows, spanProps(fmt.Sprintf("SECTION B: TRADE DETAILS (%d)", len(note.Trades)), propSpanActive, sec, color.ThemeSectionFG, rowHeightSectionHdr)) // cold path
	th := color.ThemeTableHead
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightSub,
		Cells: []layout.Cell{
			layout.StyledCell("Symbol", font, fontSizeTableCell, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Action", font, fontSizeTableCell, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Qty", font, fontSizeTableCell, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Price", font, fontSizeTableCell, color.ThemeBlack, &th, 0, rowHeightSub),
			layout.StyledCell("Total", font, fontSizeTableCell, color.ThemeBlack, &th, 0, rowHeightSub),
		},
	})
	trades := note.Trades
	var tradeBuf []byte
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
		tradeBuf = strconv.AppendInt(tradeBuf[:0], int64(t.Qty), decimalBase)
		tl.Rows = append(tl.Rows, layout.Row{
			Height: rowHeightTradeHFT,
			Cells: []layout.Cell{
				{Text: t.Symbol, Style: cs, W: 0, H: rowHeightTradeHFT},
				{Text: t.Action, Style: ca, W: 0, H: rowHeightTradeHFT},
				{Text: string(tradeBuf), Style: cs, W: 0, H: rowHeightTradeHFT},
				{Text: model.Money(t.Price), Style: cs, W: 0, H: rowHeightTradeHFT},
				{Text: model.Money(t.Total), Style: cs, W: 0, H: rowHeightTradeHFT},
			},
		})
	}
	tl.Rows = append(tl.Rows, spanProps("SECTION C: SUMMARY", propSpanActive, sec, color.ThemeSectionFG, rowHeightSectionHdr))
	turnover, brok, reg := 0.0, 20.0, 150.0
	if note.Summary != nil {
		turnover = note.Summary.TotalTurnover
		brok = note.Summary.Brokerage
		reg = note.Summary.RegulatoryCharges
	}
	sumBG := color.ThemeSummaryBG
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightSummary,
		Cells: []layout.Cell{
			layout.StyledCell("Turnover", font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell(model.Money(turnover), font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell("Charges", font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell(model.Money(brok+reg), font, fontSizeSummaryLabel, color.ThemeBlack, nil, 0, rowHeightSummary),
			layout.StyledCell(model.Money(turnover+brok+reg), font, fontSizeSummaryLabel, color.ThemeBlack, &sumBG, 0, rowHeightSummary),
		},
	})
	return tl
}

func buildHFT(note *model.ContractNote) *layout.TableLayout {
	cols := []float64{2, 1, 2, 0.8, 0.6, 2, 1}
	tl := &layout.TableLayout{ColWidths: cols}
	cell7h := layout.CellStyleFromColors(font, fontSizeSectionLabel, color.ThemeBlack, nil, defaultCellPadding)
	cell7h.Border = layout.DefaultBorder()

	bgH := color.ThemeHeaderBG
	dateStr := time.Now().Format("2006-01-02")
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightHeader,
		Cells: []layout.Cell{
			layout.StyledCell("HFT CONTRACT NOTE", font, fontSizeHeaderHFT, color.ThemeHeaderFG, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell("", font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell(note.Client.Name, font, fontSizeTableCellHFT, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader),
			layout.StyledCell(fmt.Sprintf("%d Trades | %s", len(note.Trades), dateStr), font, fontSizeHeaderSub, color.ThemeHeaderSub, &bgH, 0, rowHeightHeader), // cold path
		},
	})

	sec := color.ThemeSectionBG
	tl.Rows = append(tl.Rows, spanProps("SECTION A: CLIENT INFORMATION", propSpanHFT, sec, color.ThemeSectionFG, rowHeightSub))
	info := color.ThemeInfoRow
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightAudit,
		Cells: []layout.Cell{
			layout.StyledCell("Client", font, fontSizeTableCell, color.ThemeBlack, &info, 0, rowHeightAudit),
			layout.StyledCell(note.Client.Name, font, fontSizeTableCell, color.ThemeBlack, &info, 0, rowHeightAudit),
			layout.StyledCell("Code", font, fontSizeTableCell, color.ThemeBlack, &info, 0, rowHeightAudit),
			layout.StyledCell(note.Client.Code, font, fontSizeTableCell, color.ThemeBlack, &info, 0, rowHeightAudit),
			layout.StyledCell("PAN", font, fontSizeTableCell, color.ThemeBlack, &info, 0, rowHeightAudit),
			layout.StyledCell(note.Client.PAN, font, fontSizeTableCell, color.ThemeBlack, &info, 0, rowHeightAudit),
			layout.StyledCell("BATCH", font, fontSizeTableCell, color.ThemeBlack, &info, 0, rowHeightAudit),
		},
	})
	tl.Rows = append(tl.Rows, spanProps(fmt.Sprintf("SECTION B: TRADES (%d)", len(note.Trades)), propSpanHFT, sec, color.ThemeSectionFG, rowHeightSub)) // cold path
	th := color.ThemeTableHead
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightAudit,
		Cells: []layout.Cell{
			layout.StyledCell("ID", font, fontSizeSectionLabel, color.ThemeBlack, &th, 0, rowHeightAudit),
			layout.StyledCell("Time", font, fontSizeSectionLabel, color.ThemeBlack, &th, 0, rowHeightAudit),
			layout.StyledCell("Symbol", font, fontSizeSectionLabel, color.ThemeBlack, &th, 0, rowHeightAudit),
			layout.StyledCell("Action", font, fontSizeSectionLabel, color.ThemeBlack, &th, 0, rowHeightAudit),
			layout.StyledCell("Qty", font, fontSizeSectionLabel, color.ThemeBlack, &th, 0, rowHeightAudit),
			layout.StyledCell("Price", font, fontSizeSectionLabel, color.ThemeBlack, &th, 0, rowHeightAudit),
			layout.StyledCell("Total", font, fontSizeSectionLabel, color.ThemeBlack, &th, 0, rowHeightAudit),
		},
	})
	trades := note.Trades
	var tradeBuf []byte
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
		tradeBuf = strconv.AppendInt(tradeBuf[:0], int64(t.ID), decimalBase)
		idStr := string(tradeBuf)
		tradeBuf = strconv.AppendInt(tradeBuf[:0], int64(t.Qty), decimalBase)
		tl.Rows = append(tl.Rows, layout.Row{
			Height: rowHeightTradeSm,
			Cells: []layout.Cell{
				{Text: idStr, Style: cs, W: 0, H: rowHeightTradeSm},
				{Text: t.Time, Style: cs, W: 0, H: rowHeightTradeSm},
				{Text: t.Symbol, Style: cs, W: 0, H: rowHeightTradeSm},
				{Text: t.Action, Style: ca, W: 0, H: rowHeightTradeSm},
				{Text: string(tradeBuf), Style: cs, W: 0, H: rowHeightTradeSm},
				{Text: model.Money(t.Price), Style: cs, W: 0, H: rowHeightTradeSm},
				{Text: model.Money(t.Total), Style: cs, W: 0, H: rowHeightTradeSm},
			},
		})
	}
	tl.Rows = append(tl.Rows, spanProps("SECTION C: COMPLIANCE AUDIT", propSpanHFT, sec, color.ThemeSectionFG, rowHeightSub))
	ts := "2024-02-12T17:00:00Z"
	if note.Audit != nil && note.Audit.Timestamp != "" {
		ts = note.Audit.Timestamp
	}
	tl.Rows = append(tl.Rows, layout.Row{
		Height: rowHeightAudit,
		Cells: []layout.Cell{
			layout.StyledCell("Audit", font, fontSizeTableCell, color.ThemeBlack, nil, 0, rowHeightAudit),
			layout.StyledCell(ts, font, fontSizeTableCell, color.ThemeBlack, nil, 0, rowHeightAudit),
			layout.StyledCell("Signature", font, fontSizeTableCell, color.ThemeBlack, nil, 0, rowHeightAudit),
			layout.StyledCell("[Placeholder]", font, fontSizeTableCell, color.ThemeBlack, nil, 0, rowHeightAudit),
			layout.StyledCell("", font, fontSizeTableCell, color.ThemeBlack, nil, 0, rowHeightAudit),
			layout.StyledCell("", font, fontSizeTableCell, color.ThemeBlack, nil, 0, rowHeightAudit),
			layout.StyledCell("", font, fontSizeTableCell, color.ThemeBlack, nil, 0, rowHeightAudit),
		},
	})
	return tl
}
