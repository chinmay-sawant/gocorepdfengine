# gocorepdfengine — Phase Plans

Source: [baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md](./baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md)

Each phase is a **checklist plan** you can execute independently. Complete phases in order unless noted.

| Phase | File | Goal | Gate |
|-------|------|------|------|
| 1 | [phase-01-core-pdf20-writer.md](./phase-01-core-pdf20-writer.md) | Minimal PDF 2.0 shell | Unit / open in viewer |
| 2 | [phase-02-layout-primitives.md](./phase-02-layout-primitives.md) | Text, tables, multi-page, images | Visual fixtures |
| 3 | [phase-03-font-embedding.md](./phase-03-font-embedding.md) | TTF subset + Type0 embed | Glyph/width tests |
| 4 | [phase-04-pdfa4-compliance.md](./phase-04-pdfa4-compliance.md) | PDF/A-4 archival objects | **veraPDF `-f 4`** |
| 5 | [phase-05-pdfua2-tagging.md](./phase-05-pdfua2-tagging.md) | PDF/UA-2 structure tree | **veraPDF `-f ua2`** |
| 6 | [phase-06-performance-pooling.md](./phase-06-performance-pooling.md) | Speed / memory parity | Bench (after 4+5 green) |
| 7 | [phase-07-optional-product-features.md](./phase-07-optional-product-features.md) | Sign, encrypt, forms | Separate product gates |

**Default compliant profile (end of phase 5):** PDF 2.0 + PDF/A-4 + PDF/UA-2.

**Out of scope across all phases:** HTTP API, frontend, bindings, merge/redact product surface.

## Compliance harness (ready now)

Scripts live under [`../compliance/`](../compliance/) (ported from gopdfsuit):

```bash
make install-verapdf          # project-local veraPDF CLI
make install-pdf-validators   # + avalpdf
make test-verify-pdfs         # PDF/A-4 + PDF/UA-2 on compliance/fixtures/
```

See [`../compliance/README.md`](../compliance/README.md). Put golden/generated PDFs in `compliance/fixtures/`.
