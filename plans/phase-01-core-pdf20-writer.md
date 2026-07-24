# Phase 1 — Core PDF 2.0 Writer

**Status:** Not started  
**Depends on:** None  
**Base plan refs:** §4 (`doc`, `write`, `page`), §5 Phase 0–1/5–6 (minimal), §6.1–6.3, §6.6–6.7, §11 Phase A

---

## Goal

Ship a minimal **untagged** PDF 2.0 file: valid header, catalog, pages tree, one page, simple content stream, one font reference, classic xref, trailer with `/ID`.

---

## Package layout (scaffold this phase)

- [ ] `engine/doc/` — object ID allocator, document builder, final assembly
- [ ] `engine/write/` — dict/stream/xref/trailer byte encoding
- [ ] `engine/page/` — pages tree + single page object
- [ ] `engine/content/` — minimal text operators only
- [ ] `go.mod` + package smoke test

---

## Checklist — document shell

### Header
- [ ] Emit `%PDF-2.0`
- [ ] Emit binary comment line (bytes ≥ 128)

### Catalog (`/Type /Catalog`)
- [ ] `/Type /Catalog`
- [ ] `/Pages` → pages root

### Pages tree
- [ ] `/Type /Pages`
- [ ] `/Kids [ <page> 0 R ]`
- [ ] `/Count 1`

### Page
- [ ] `/Type /Page`
- [ ] `/Parent` pages root
- [ ] `/MediaBox [0 0 width height]` (e.g. A4)
- [ ] `/Contents` content stream ref
- [ ] `/Resources /Font` with at least one font name

### Content stream
- [ ] Stream object with `/Length`
- [ ] Simple text operators (e.g. `BT` … `ET`) drawing one line
- [ ] Optional: `/Filter /FlateDecode` (can defer to later)

### Font (placeholder only)
- [ ] Simple Type1 standard font dict is OK for this phase (e.g. Helvetica)
- [ ] Do **not** require embedding yet (phase 3)

### XRef + trailer
- [ ] Classic `xref` with correct byte offsets
- [ ] `trailer` with `/Size`, `/Root`, `/ID [ <file-id> <file-id> ]`
- [ ] Optional non-A `/Info` with `/CreationDate` / `/ModDate` (allowed until phase 4)
- [ ] `startxref` + `%%EOF`

### Mode hooks (stubs only)
- [ ] Document mode flags exist: `ModePDF20` (always on), stubs for `ModePDFA4`, `ModePDFUA2`, `ModeEmbedFonts`
- [ ] No A-4 / UA-2 objects yet

---

## Acceptance criteria

- [ ] Generated file opens in at least one PDF viewer
- [ ] Header starts with `%PDF-2.0`
- [ ] Unit test: every object ID in xref resolves to a real `n` offset
- [ ] Unit test: trailer `/Root` points at catalog
- [ ] Unit test: `/ID` present with two entries

---

## Explicitly out of scope

- [ ] Tables, multi-page, images
- [ ] Font embedding / subsetting
- [ ] XMP, OutputIntent, ICC
- [ ] Structure tree / BDC-EMC
- [ ] veraPDF (not required until phase 4)

---

## Done when

One API (e.g. `GenerateMinimalPDF`) returns `[]byte` for a single-page “Hello” PDF 2.0 with valid xref.
