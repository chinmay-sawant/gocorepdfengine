# gocorepdfengine — build, test, and PDF compliance validation

# ── PDF validators (ported from gopdfsuit) ───────────────────────────────────
# Install:  make install-verapdf | make install-pdf-validators
# Validate: make test-verify-pdfs
# Paths:    compliance/  (scripts)  verapdf/  (CLI, gitignored)
#           compliance/fixtures/  (PDFs under test)
# Env:      VERAPDF_BIN  AVALPDF_BIN  VERIFY_PDFS_JOBS  COMPLIANCE_FLAVOURS
#           VERIFY_STRUCTURE_TREE  VERIFY_AVALPDF  VERIFY_AVALPDF_STRICT

.PHONY: help \
	install-verapdf install-pdf-validators \
	test-verify-pdfs test-scan-pdfs test-scan-pdfs-compliance test-compliance \
	test test-unit clean

help:
	@echo "gocorepdfengine targets"
	@echo ""
	@echo "  Validators:"
	@echo "    make install-verapdf           Install project-local veraPDF CLI"
	@echo "    make install-pdf-validators    veraPDF + avalpdf (venv)"
	@echo "    make test-verify-pdfs          PDF/A-4 + PDF/UA-2 on compliance/fixtures/"
	@echo "    make test-scan-pdfs            Parse-only validity scan of fixtures"
	@echo "    make test-scan-pdfs-compliance Scan + full PDF/A-4 and PDF/UA-2 table"
	@echo "    make test-compliance           Alias for test-verify-pdfs"
	@echo ""
	@echo "  Engine (placeholder until packages exist):"
	@echo "    make test-unit                 go test ./..."
	@echo "    make test                      unit + compliance (when fixtures exist)"
	@echo "    make clean                     remove bin/"
	@echo ""
	@echo "  Single-PDF helper:"
	@echo "    ./compliance/run_verapdf.sh --both path/to/file.pdf"
	@echo "    ./compliance/verify_pdfs.sh --pdf path/to/file.pdf"

install-verapdf:
	bash compliance/install_verapdf.sh

install-pdf-validators:
	bash compliance/install_pdf_validators.sh

test-verify-pdfs:
	bash compliance/verify_pdfs.sh

test-scan-pdfs:
	bash compliance/verify_pdfs.sh --scan-all

test-scan-pdfs-compliance:
	bash compliance/verify_pdfs.sh --scan-all-compliance

test-compliance: test-verify-pdfs

# Unit tests only (skip if no Go modules yet)
test-unit:
	@if [ -f go.mod ]; then \
		go test ./...; \
	else \
		echo "SKIP test-unit: go.mod not present yet"; \
	fi

# Full gate: unit tests, then compliance if fixtures exist
test: test-unit
	@pdf_count=$$(find compliance/fixtures -type f -iname '*.pdf' 2>/dev/null | wc -l); \
	if [ "$$pdf_count" -eq 0 ]; then \
		echo "SKIP compliance: no PDFs in compliance/fixtures yet"; \
	else \
		bash compliance/verify_pdfs.sh; \
	fi

clean:
	rm -rf bin/
