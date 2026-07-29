// Package main generates a financial report PDF from the full-format JSON template.
//
// Usage:
//
//	cd sampledata/financial && go run .
//	# or from project root:
//	go run ./sampledata/financial
package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/chinmay/gocorepdfengine/engine/model"
	"github.com/chinmay/gocorepdfengine/engine/render"
)

const outputFilePerm = 0o600

func main() {
	// Resolve JSON path relative to the project root.
	// codehound-ignore: BP-1
	pwd, _ := os.Getwd() // Getwd error discarded; path is a best-effort fallback for sample data.
	path := filepath.Join(pwd, "sampledata/financial/financial_report.json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = filepath.Join(pwd, "financial_report.json")
	}
	t, err := model.LoadTemplate(path)
	if err != nil {
		panic(err)
	}

	pdf, err := render.TemplatePDF(t, render.Options{Compliant: false})
	if err != nil {
		panic(err)
	}

	outPath := filepath.Join(filepath.Dir(path), "financial_report_output.pdf")
	if err := os.WriteFile(outPath, pdf, outputFilePerm); err != nil {
		panic(err)
	}
	fmt.Printf("Saved: %s (%d bytes)\n", outPath, len(pdf))
}
