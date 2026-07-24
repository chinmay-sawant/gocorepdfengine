// Package model holds document models loaded from JSON templates
// (sampledata/zerodha/*.json) and expanded for benchmarks.
package model

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strconv"
)

// ContractNote is the domain model for Zerodha-style contract notes.
type ContractNote struct {
	DocumentType string `json:"document_type"`

	Compliance *Compliance `json:"compliance,omitempty"`
	Metadata   *Metadata   `json:"metadata,omitempty"`
	Features   *Features   `json:"features,omitempty"`
	Footer     *Footer     `json:"footer,omitempty"`

	Client Client  `json:"client"`
	Trades []Trade `json:"trades"`

	Financials *Financials `json:"financials,omitempty"`
	Summary    *Summary    `json:"summary,omitempty"`
	Audit      *Audit      `json:"compliance_audit,omitempty"`

	// Runtime-only (not always in JSON).
	Title     string `json:"-"`
	Watermark string `json:"-"`
	ModeLabel string `json:"-"` // retail | active | hft
}

type Compliance struct {
	PDFStandard   string `json:"pdf_standard"`
	Accessibility string `json:"accessibility"`
	ICCProfile    string `json:"icc_profile"`
}

type Metadata struct {
	Title    string   `json:"title"`
	Author   string   `json:"author"`
	Subject  string   `json:"subject"`
	Keywords []string `json:"keywords"`
}

type Features struct {
	Bookmarks     bool   `json:"bookmarks"`
	InternalLinks bool   `json:"internal_links"`
	Watermark     string `json:"watermark"`
}

type Footer struct {
	Font string `json:"font"`
	Text string `json:"text"`
	Link string `json:"link,omitempty"`
}

type Client struct {
	Name string `json:"name"`
	Code string `json:"code"`
	PAN  string `json:"pan"`
}

type Trade struct {
	ID       int     `json:"id,omitempty"`
	Time     string  `json:"time,omitempty"`
	Symbol   string  `json:"symbol"`
	ISIN     string  `json:"isin,omitempty"`
	Action   string  `json:"action"`
	Qty      int     `json:"qty"`
	Price    float64 `json:"price"`
	Total    float64 `json:"total"`
	Currency string  `json:"currency,omitempty"`
}

type Financials struct {
	NetObligation float64 `json:"net_obligation"`
	STTTax        float64 `json:"stt_tax"`
}

type Summary struct {
	TotalTurnover      float64 `json:"total_turnover"`
	Brokerage          float64 `json:"brokerage"`
	RegulatoryCharges  float64 `json:"regulatory_charges"`
}

type Audit struct {
	Timestamp                   string `json:"timestamp"`
	AuditorSignaturePlaceholder bool   `json:"auditor_signature_placeholder"`
}

// LoadJSON reads a contract-note template from disk.
func LoadJSON(path string) (*ContractNote, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var note ContractNote
	if err := json.Unmarshal(data, &note); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err) // cold path (one-time load)
	}
	note.applyDefaults()
	return &note, nil
}

func (n *ContractNote) applyDefaults() {
	switch n.DocumentType {
	case "contract_note_retail":
		n.ModeLabel = "retail"
		if n.Title == "" && n.Metadata != nil {
			n.Title = n.Metadata.Title
		}
		if n.Title == "" {
			n.Title = "Contract Note - CN2024001"
		}
	case "contract_note_active":
		n.ModeLabel = "active"
		n.Title = "Active Trader Contract Note"
		if n.Features != nil {
			n.Watermark = n.Features.Watermark
		}
	case "contract_note_hft":
		n.ModeLabel = "hft"
		n.Title = "HFT Contract Note - Algo Capital LLP"
	default:
		n.ModeLabel = "unknown"
		if n.Title == "" {
			n.Title = "Contract Note"
		}
	}
}

// Clone shallow-copies the note (trades slice shared until ExpandTrades replaces it).
func (n *ContractNote) Clone() *ContractNote {
	if n == nil {
		return nil
	}
	cp := *n
	if n.Trades != nil {
		cp.Trades = append([]Trade(nil), n.Trades...)
	}
	return &cp
}

var symbols = []string{
	"RELIANCE", "TCS", "INFY", "HDFCBANK", "TATASTEEL",
	"ICICIBANK", "SBIN", "WIPRO", "BHARTIARTL", "LT",
	"NIFTY24FEB22000CE", "NIFTY24FEB22000PE",
	"BANKNIFTY24FEB46000CE", "BANKNIFTY24FEB46000PE",
	"AXISBANK", "KOTAKBANK", "MARUTI", "TITAN", "ADANIENT", "BAJFINANCE",
}

// ExpandTrades replaces trades with n synthetic rows (active=40, hft=2000).
// Retail keeps JSON trades when n <= 0 or n == len(existing) for small sets.
func (n *ContractNote) ExpandTrades(count int, seed int64) {
	if count <= 0 {
		return
	}
	// Keep retail JSON trades if count matches or is small and already present.
	if n.ModeLabel == "retail" && len(n.Trades) > 0 && count <= len(n.Trades) {
		return
	}
	rng := rand.New(rand.NewSource(seed)) // Deterministic seed for benchmark reproducibility (not security-sensitive).
	trades := make([]Trade, count)
	hour, mn, sec := 9, 15, 0
	symCount := len(symbols) //nolint: perflint // PERF-109: cached len for loop
	for i := 0; i < count; i++ {
		sym := symbols[rng.Intn(symCount)]
		action := "BUY"
		if rng.Intn(2) == 1 {
			action = "SELL"
		}
		qty := (rng.Intn(50) + 1) * 10
		price := 100.0 + rng.Float64()*3400.0
		price = float64(int(price*100)) / 100
		total := float64(qty) * price

		timeStr := fmt.Sprintf("%02d:%02d:%02d", hour, mn, sec) //nolint: perflint // PERF-6: test data generation (cold path)
		sec++
		if sec >= 60 {
			sec = 0
			mn++
		}
		if mn >= 60 {
			mn = 0
			hour++
		}

		trades[i] = Trade{
			ID:     i + 1,
			Time:   timeStr,
			Symbol: sym,
			Action: action,
			Qty:    qty,
			Price:  price,
			Total:  total,
		}
	}
	n.Trades = trades

	if n.ModeLabel == "active" || n.Summary != nil {
		var turnover float64
		for _, t := range trades {
			turnover += t.Total
		}
		if n.Summary == nil {
			n.Summary = &Summary{}
		}
		n.Summary.TotalTurnover = turnover
		if n.Summary.Brokerage == 0 {
			n.Summary.Brokerage = 20
		}
		if n.Summary.RegulatoryCharges == 0 {
			n.Summary.RegulatoryCharges = 150
		}
	}
}

// Money formats a rupee amount for display (ASCII "Rs." for portable fonts).
func Money(v float64) string {
	return "Rs." + strconv.FormatFloat(v, 'f', 2, 64)
}
