package engine

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

func TestGenerateMinimalPDF(t *testing.T) {
	result, err := Generate(Config{
		Width:   595.276,
		Height:  841.89,
		Font:    "Helvetica",
		FontSize: 12,
		Text:    "Hello, PDF 2.0!",
	})
	if err != nil {
		t.Fatal(err)
	}
	data := result.Data

	if !bytes.HasPrefix(data, []byte("%PDF-2.0")) {
		t.Error("header must start with %PDF-2.0")
	}

	if !bytes.Contains(data, []byte("xref")) {
		t.Error("missing xref section")
	}

	trimmed := bytes.TrimSpace(data)
	if !bytes.HasSuffix(trimmed, []byte("%%EOF")) {
		t.Error("must end with percent-percent-EOF")
	}

	if !bytes.Contains(data, []byte("/Type /Catalog")) {
		t.Error("missing /Type /Catalog")
	}
	if !bytes.Contains(data, []byte("/Type /Pages")) {
		t.Error("missing /Type /Pages")
	}
	if !bytes.Contains(data, []byte("/Type /Page")) {
		t.Error("missing /Type /Page")
	}

	if !bytes.Contains(data, []byte("/Root")) {
		t.Error("missing /Root in trailer")
	}
	if !bytes.Contains(data, []byte("/ID")) {
		t.Error("missing /ID in trailer")
	}

	lines := strings.Split(string(data), "\n")
	foundEndobj := 0
	for _, line := range lines {
		if strings.TrimSpace(line) == "endobj" {
			foundEndobj++
		}
	}
	if foundEndobj < 4 {
		t.Errorf("expected at least 4 endobj markers, got %d", foundEndobj)
	}

	path := filepath.Join(t.TempDir(), "test_output.pdf")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("wrote %s (%d bytes)", path, len(data))
}

func TestGeneratePDFA4(t *testing.T) {
	result, err := Generate(Config{
		Width:   595.276,
		Height:  841.89,
		Font:    "Helvetica",
		FontSize: 12,
		Text:    "PDF/A-4 test",
		Mode:    doc.ModePDFA4,
	})
	if err != nil {
		t.Fatal(err)
	}
	data := result.Data

	if !bytes.HasPrefix(data, []byte("%PDF-2.0")) {
		t.Error("header must start with %PDF-2.0")
	}
	if !bytes.Contains(data, []byte("/OutputIntents")) {
		t.Error("missing /OutputIntents")
	}
	if !bytes.Contains(data, []byte("/Metadata")) {
		t.Error("missing /Metadata")
	}
	if !bytes.Contains(data, []byte("ICCBased")) {
		t.Error("missing ICCBased")
	}
	if !bytes.Contains(data, []byte("/DefaultRGB")) {
		t.Error("missing /DefaultRGB")
	}
	// Trailer must not contain /Info (PDF/A-4 forbids document info dict)
	trailerStart := bytes.LastIndex(data, []byte("trailer"))
	if trailerStart >= 0 {
		trailerSection := data[trailerStart:]
		if bytes.Contains(trailerSection, []byte("/Info")) {
			t.Error("trailer should not contain /Info in A-4 mode")
		}
	}
}

func TestGeneratePDFUA2(t *testing.T) {
	result, err := Generate(Config{
		Width:   595.276,
		Height:  841.89,
		Font:    "Helvetica",
		FontSize: 12,
		Text:    "PDF/UA-2 test",
		Mode:    doc.ModePDFUA2,
		Lang:    "en-US",
	})
	if err != nil {
		t.Fatal(err)
	}
	data := result.Data

	if !bytes.HasPrefix(data, []byte("%PDF-2.0")) {
		t.Error("header must start with %PDF-2.0")
	}
	if !bytes.Contains(data, []byte("/MarkInfo")) {
		t.Error("missing /MarkInfo")
	}
	if !bytes.Contains(data, []byte("/Marked true")) {
		t.Error("missing /Marked true")
	}
	if !bytes.Contains(data, []byte("/StructTreeRoot")) {
		t.Error("missing /StructTreeRoot")
	}
	if !bytes.Contains(data, []byte("/Lang")) {
		t.Error("missing /Lang")
	}
	if !bytes.Contains(data, []byte("/ViewerPreferences")) {
		t.Error("missing /ViewerPreferences")
	}
	if !bytes.Contains(data, []byte("/DisplayDocTitle")) {
		t.Error("missing /DisplayDocTitle")
	}
	if !bytes.Contains(data, []byte("/StructParents 0")) {
		t.Error("missing /StructParents 0")
	}
	if !bytes.Contains(data, []byte("/Tabs /S")) {
		t.Error("missing /Tabs /S")
	}
}

func TestGeneratePDFA4WithUA2(t *testing.T) {
	result, err := Generate(Config{
		Width:   595.276,
		Height:  841.89,
		Font:    "Helvetica",
		FontSize: 12,
		Text:    "PDF/A-4 + UA-2 test",
		Mode:    doc.ModePDFA4 | doc.ModePDFUA2,
		Lang:    "en-US",
		Title:   "Combined Test",
		Author:  "Test Author",
	})
	if err != nil {
		t.Fatal(err)
	}
	data := result.Data

	if !bytes.Contains(data, []byte("/OutputIntents")) {
		t.Error("missing /OutputIntents")
	}
	if !bytes.Contains(data, []byte("/StructTreeRoot")) {
		t.Error("missing /StructTreeRoot")
	}
	if !bytes.Contains(data, []byte("/MarkInfo")) {
		t.Error("missing /MarkInfo")
	}
	trailerStart := bytes.LastIndex(data, []byte("trailer"))
	if trailerStart >= 0 {
		trailerSection := data[trailerStart:]
		if bytes.Contains(trailerSection, []byte("/Info")) {
			t.Error("trailer should not contain /Info in combined A-4+UA mode")
		}
	}
}
