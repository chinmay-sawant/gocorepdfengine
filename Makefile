# gocorepdfengine — build, test, PDF compliance, Zerodha-style bench

# ── Benchmark defaults ───────────────────────────────────────────────────────
GO_BENCH ?= go
GOMAXPROCS_BENCH ?= 24
BENCH_ITERATIONS ?= 5000
BENCH_WORKERS ?= 48
BENCH_CACHE ?= 1
ZERODHA_DIR := sampledata/zerodha

.PHONY: help \
	install-verapdf install-pdf-validators \
	test-verify-pdfs test-structure-tree test-scan-pdfs test-scan-pdfs-compliance test-compliance \
	test test-unit clean \
	bench-help \
	bench-zerodha bench-zerodha-nocomply bench-zerodha-nocomply-x10 \
	bench-zerodha-x2 bench-zerodha-x5 bench-zerodha-x10 bench-zerodha-x10-pprof \
	bench-zerodha-cached bench-zerodha-uncached

help:
	@echo "gocorepdfengine targets"
	@echo ""
	@echo "  Validators:"
	@echo "    make install-verapdf | install-pdf-validators"
	@echo "    make test-verify-pdfs | test-structure-tree | test-compliance"
	@echo ""
	@echo "  Zerodha-style gold standard (sampledata/zerodha — JSON→model→layout):"
	@echo "    make bench-zerodha                 # PDF/A-4+UA-2 flags, model cache ON"
	@echo "    make bench-zerodha-nocomply        # PDF 2.0 only, cache ON"
	@echo "    make bench-zerodha-cached          # explicit BENCH_CACHE=1"
	@echo "    make bench-zerodha-uncached        # BENCH_CACHE=0 (rebuild model each iter)"
	@echo "    make bench-zerodha-nocomply-x10"
	@echo "    make bench-zerodha-x2"
	@echo "    make bench-zerodha-x5              # 5 runs + CPU/heap pprof"
	@echo "    make bench-zerodha-x10"
	@echo "    make bench-zerodha-x10-pprof"
	@echo ""
	@echo "  Overrides: GO_BENCH GOMAXPROCS_BENCH BENCH_ITERATIONS BENCH_WORKERS BENCH_CACHE"
	@echo "             BENCH_SKIP_WRITE BENCH_WARMUP BENCH_SEED"
	@echo ""
	@echo "  Engine: make test-unit | make test | make clean"

bench-help: help

install-verapdf:
	bash compliance/install_verapdf.sh

install-pdf-validators:
	bash compliance/install_pdf_validators.sh

test-verify-pdfs:
	bash compliance/verify_pdfs.sh

test-structure-tree:
	@pdfs=$$(find compliance/fixtures -type f -iname '*.pdf' 2>/dev/null | sort); \
	if [ -z "$$pdfs" ]; then echo "SKIP: no PDFs in compliance/fixtures"; \
	else python3 compliance/structure_tree_check.py $$pdfs; fi

test-scan-pdfs:
	bash compliance/verify_pdfs.sh --scan-all

test-scan-pdfs-compliance:
	bash compliance/verify_pdfs.sh --scan-all-compliance

test-compliance: test-verify-pdfs test-structure-tree

test-unit:
	go test ./...

test: test-unit
	@pdf_count=$$(find compliance/fixtures -type f -iname '*.pdf' 2>/dev/null | wc -l); \
	if [ "$$pdf_count" -eq 0 ]; then echo "SKIP compliance: no fixtures"; \
	else bash compliance/verify_pdfs.sh; fi

clean:
	rm -rf bin/

# ── Zerodha-style bench (local engine only) ──────────────────────────────────

bench-zerodha:
	cd $(ZERODHA_DIR) && GOMAXPROCS=$(GOMAXPROCS_BENCH) BENCH_ITERATIONS=$(BENCH_ITERATIONS) \
		BENCH_WORKERS=$(BENCH_WORKERS) BENCH_CACHE=$(BENCH_CACHE) $(GO_BENCH) run .

bench-zerodha-nocomply:
	cd $(ZERODHA_DIR) && GOMAXPROCS=$(GOMAXPROCS_BENCH) BENCH_ITERATIONS=$(BENCH_ITERATIONS) \
		BENCH_WORKERS=$(BENCH_WORKERS) BENCH_CACHE=$(BENCH_CACHE) $(GO_BENCH) run -tags nocomply .

bench-zerodha-cached:
	@$(MAKE) bench-zerodha BENCH_CACHE=1

bench-zerodha-uncached:
	@$(MAKE) bench-zerodha BENCH_CACHE=0

bench-zerodha-nocomply-x10:
	bash $(ZERODHA_DIR)/run_bench_x10_nocomply.sh

bench-zerodha-x2:
	@for i in 1 2; do echo "=== zerodha run $$i / 2 ==="; $(MAKE) bench-zerodha; done

bench-zerodha-x5:
	bash $(ZERODHA_DIR)/run_bench_x5.sh

bench-zerodha-x10:
	bash $(ZERODHA_DIR)/run_bench_x10.sh

bench-zerodha-x10-pprof: bench-zerodha-x10 bench-zerodha-x5

# ── Full-template financial report ────────────────────────────────────────────

FINANCIAL_DIR := sampledata/financial

bench-financial:
	cd $(FINANCIAL_DIR) && go run .
