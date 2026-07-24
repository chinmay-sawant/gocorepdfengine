# Phase 2 — Layout Primitives (Untagged)

**Status:** Not started  
**Depends on:** Phase 1 complete  
**Base plan refs:** §4 (`layout`, `image`, `content`), §5 Phase 2–3 (untagged), §11 Phase B

---

## Goal

Add real drawing: text runs, tables, borders, multi-page flow, and raster images — still **untagged**, still **no PDF/A**.

---

## Package layout

- [ ] `engine/layout/` — margins, flow, tables, page break
- [ ] `engine/image/` — JPEG/PNG decode → XObject (DeviceRGB OK)
- [ ] Extend `engine/content/` — path operators, text state, `Do` for images
- [ ] Extend `engine/page/` — multi-page kids, per-page content streams

---

## Checklist — text & geometry

- [ ] Positioned text with font size / color
- [ ] Text escape for PDF string literals `(…)`
- [ ] Path borders: `m` / `l` / `re` / `S` / `f`
- [ ] Page margins and content box
- [ ] Multi-page: content overflow creates new page objects + updates `/Count`

## Checklist — tables

- [ ] Fixed-column table grid
- [ ] Cell text with wrapping (basic)
- [ ] Cell borders (vector)
- [ ] Header row styling (visual only; no `/TH` structure yet)
- [ ] Table can span pages (row split or page-break between rows)

## Checklist — images

- [ ] JPEG → `/Subtype /Image` with `/Filter /DCTDecode`
- [ ] PNG → decode RGB, `/Filter /FlateDecode`
- [ ] XObject keys: `/Type /XObject`, `/Subtype /Image`, `/Width`, `/Height`, `/ColorSpace /DeviceRGB` (or gray), `/BitsPerComponent`, `/Length`
- [ ] Image drawn via content `cm` + `/ImN Do`
- [ ] Optional: dedupe identical image bytes → one XObject

## Checklist — resources

- [ ] Per-page `/Resources` with `/Font` and `/XObject` maps
- [ ] Content stream compression with `/Filter /FlateDecode` (recommended)

## Checklist — fixtures

- [ ] Fixture: single page title + paragraph
- [ ] Fixture: 3×3 table
- [ ] Fixture: multi-page long table
- [ ] Fixture: page with embedded JPEG and/or PNG

---

## Acceptance criteria

- [ ] All fixtures open cleanly in a viewer
- [ ] Multi-page `/Count` matches actual pages
- [ ] Images visible and correctly sized
- [ ] Manual visual parity vs simple gopdfsuit templates (spot-check)

---

## Explicitly out of scope

- [ ] BDC / EMC / structure tree
- [ ] ICCBased color / OutputIntent
- [ ] Liberation embedding (use standard fonts or phase-3 prep only)
- [ ] veraPDF gates

---

## Done when

Engine can emit multi-page documents with text, tables, borders, and images without compliance claims.
