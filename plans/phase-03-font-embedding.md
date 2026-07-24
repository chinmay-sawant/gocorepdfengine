# Phase 3 — Font Embedding Pipeline

**Status:** Not started  
**Depends on:** Phase 1 complete (phase 2 recommended)  
**Base plan refs:** §4 (`font`), §7.5, §11 Phase C, Liberation map in base plan

---

## Goal

Embed TrueType fonts as PDF CID fonts with correct subsetting so phase 4 can claim PDF/A-4 without external fonts.

---

## Package layout

- [ ] `engine/font/` — TTF load, metrics, subset, registry, Liberation map
- [ ] Integration with `doc` emit order: assign font object IDs **before** content if content refs need stable names; subsets **after** content char collection

---

## Checklist — font registry

- [ ] Per-generation registry (usage tracking isolated per PDF)
- [ ] Register TTF from file path
- [ ] Register TTF from bytes / base64 (optional)
- [ ] Track used characters / glyphs during content generation
- [ ] `GenerateSubsets()` after all content is known

## Checklist — Liberation manager (PDF/A prep)

- [ ] Map standard faces → Liberation:
  - [ ] Helvetica* → LiberationSans-*
  - [ ] Times-* → LiberationSerif-*
  - [ ] Courier* → LiberationMono-*
- [ ] Ensure fonts available (local path / download policy)
- [ ] Register Liberation for all standard fonts used in a document

## Checklist — PDF object emit (CID chain)

### Type0 font
- [ ] `/Type /Font`
- [ ] `/Subtype /Type0`
- [ ] `/BaseFont`
- [ ] `/Encoding /Identity-H`
- [ ] `/DescendantFonts [ … ]`
- [ ] `/ToUnicode` stream ref

### CIDFont
- [ ] `/Subtype /CIDFontType2`
- [ ] `/CIDSystemInfo` with `/Registry`, `/Ordering`, `/Supplement`
- [ ] `/FontDescriptor`
- [ ] `/DW`
- [ ] `/W` (widths array object or inline)
- [ ] `/CIDToGIDMap`

### FontDescriptor
- [ ] `/FontName`, `/Flags`, `/FontBBox`
- [ ] `/ItalicAngle`, `/Ascent`, `/Descent`, `/CapHeight`, `/StemV`, `/XHeight`
- [ ] `/FontFile2` → subset TTF stream (prefer `/Filter /FlateDecode`)

## Checklist — correctness tests

- [ ] Used glyphs present in subset (no silent `.notdef` for used chars)
- [ ] Subset glyph widths match original scaled widths
- [ ] Hyphen width sanity (LiberationSans scaled hyphen often 333)
- [ ] ToUnicode maps used CIDs to Unicode
- [ ] Page resources reference embedded font names, not bare unembedded standards when embed mode is on

## Checklist — modes

- [ ] `ModeEmbedFonts` respected
- [ ] Non-embed path still available for non-A demos (simple Type1 / Arlington-style metrics optional)

---

## Acceptance criteria

- [ ] PDF with only embedded Liberation Sans renders text correctly offline
- [ ] Unit tests pass for subset integrity and widths
- [ ] Content stream text uses the embedded font resource name

---

## Explicitly out of scope

- [ ] Full PDF/A-4 XMP / OutputIntent (phase 4)
- [ ] Structure tagging (phase 5)
- [ ] veraPDF required gate (start wiring harness placeholders only)

---

## veraPDF harness placeholder (prep only)

- [ ] Create `compliance/verapdf/README.md` with install notes (Java 11+, `VERAPDF_BIN`)
- [ ] Stub `compliance/verapdf/run_verapdf.sh` (skip if binary missing)
- [ ] Do **not** fail CI on veraPDF yet

---

## Done when

Any text PDF can embed and subset Liberation fonts with a valid Type0/CIDFontType2 chain ready for PDF/A-4.
