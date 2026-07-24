# PDF Compliance Validators

Layered validation stack for **PDF/A-4** and **PDF/UA-2**, ported from gopdfsuit and adapted for gocorepdfengine.

Place engine-generated (or golden) PDFs under `compliance/fixtures/`.

## Quick start

```bash
# Requires Java 11+
make install-verapdf              # or: make install-pdf-validators (veraPDF + avalpdf)

# After fixtures exist:
make test-verify-pdfs             # PDF/A-4 + PDF/UA-2 + structure-tree
make test-scan-pdfs               # parse-only validity table
make test-scan-pdfs-compliance    # validity + full compliance table

# Single file:
./compliance/run_verapdf.sh --both path/to/file.pdf
./compliance/verify_pdfs.sh --pdf path/to/file.pdf
```

## Layout

| Path | Role |
|------|------|
| `install_verapdf.sh` | Download/install CLI into `<repo>/verapdf/` (gitignored) |
| `install_pdf_validators.sh` | veraPDF + avalpdf venv under `.pdf-validators/` |
| `verapdf_report.py` | Run veraPDF JSON profile check; pretty PASS/FAIL |
| `structure_tree_check.py` | ParentTree TD/TH ownership (catches bugs veraPDF misses) |
| `verify_pdfs.sh` | Parallel fixture gate (validity + flavours + structure + avalpdf) |
| `run_verapdf.sh` | Thin multi-PDF wrapper for flavours `4` / `ua2` |
| `fixtures/` | PDFs under test (empty until engine emits them) |
| `pdf_validators_requirements.txt` | avalpdf pin for the venv |

## Validator stack

| Tool | PDF/A-4 | PDF/UA-2 | Notes |
|------|---------|----------|--------|
| **veraPDF** | Yes (`-f 4`) | Yes (`-f ua2`) | Primary ISO gate |
| **structure_tree_check.py** | — | structural | ParentTree must point at TD/TH, not TR |
| **avalpdf** | No | heuristics | Warnings by default |

## Makefile targets

| Target | Action |
|--------|--------|
| `make install-verapdf` | Install project-local veraPDF |
| `make install-pdf-validators` | veraPDF + avalpdf |
| `make test-verify-pdfs` | Full compliance on `fixtures/` |
| `make test-scan-pdfs` | Parse validity only |
| `make test-scan-pdfs-compliance` | Scan + A-4 + UA-2 table |
| `make test-compliance` | Alias for `test-verify-pdfs` |
| `make test` | Unit tests + compliance if fixtures present |

## Environment

| Variable | Default | Purpose |
|----------|---------|---------|
| `VERAPDF_BIN` | `<repo>/verapdf/verapdf` | veraPDF CLI path |
| `AVALPDF_BIN` | `<repo>/.pdf-validators/venv/bin/avalpdf` | avalpdf CLI |
| `VERIFY_PDFS_JOBS` | `nproc` or `4` | Parallel workers |
| `COMPLIANCE_FLAVOURS` | `4,ua2` | Flavours for verify/scan-compliance |
| `VERIFY_STRUCTURE_TREE` | `1` | Run structure_tree_check |
| `VERIFY_AVALPDF` | `1` | Run avalpdf when installed |
| `VERIFY_AVALPDF_STRICT` | `0` | Fail on avalpdf issues when `1` |
| `COMPLIANCE_FIXTURES` | `<repo>/compliance/fixtures` | Override fixtures root |
| `NO_COLOR` | unset | Disable ANSI colours |

## Direct CLI examples

```bash
verapdf/verapdf -f 4 compliance/fixtures/minimal-text.pdf
verapdf/verapdf -f ua2 compliance/fixtures/minimal-text.pdf
python3 compliance/structure_tree_check.py compliance/fixtures/*.pdf
python3 compliance/verapdf_report.py check \
  --verapdf verapdf/verapdf \
  --pdf compliance/fixtures/minimal-text.pdf \
  --flavour 4
```

## Why structure_tree_check exists

veraPDF can pass while stricter tools (PAC/Adobe) report an invalid structure tree when:

- `ParentTree[MCID]` points at a **TR** instead of the owning **TD/TH**
- TR `/Pg` does not match child TD pages on multi-page tables

`structure_tree_check.py` enforces those rules in CI.

## Relation to phase plans

| Phase | When to use this harness |
|-------|---------------------------|
| 1–3 | Optional parse-only (`--scan-all`) once PDFs exist |
| 4 | Gate: veraPDF `-f 4` on fixtures |
| 5 | Gate: `-f 4` + `-f ua2` + structure_tree_check |
| 6+ | Re-run full matrix after optimizations |

See `plans/phase-04-pdfa4-compliance.md` and `plans/phase-05-pdfua2-tagging.md`.
