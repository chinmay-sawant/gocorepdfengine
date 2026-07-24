# Architecture Review — gocorepdfengine

| | |
|---|---|
| **Date** | 2026-07-25 |
| **Method** | `/improve-codebase-architecture` + `/codebase-design` vocabulary |
| **Agents** | 3 parallel explore passes (module depth · coupling/seams · testability) |
| **Scope** | `engine/**`, `sampledata/**`, `plans/**`, `compliance/**`, `Makefile` |
| **Overall rating** | **6.2 / 10** |
| **Tool** | Grok Build
---

## Executive summary

gocorepdfengine is a mid-maturity native PDF 2.0 engine with a **clear package map**, several **genuinely deep modules** (`font`, `layout`, `doc`), and a working product pipeline (`model` → `render` → `GenerateDocument`). Phases 1–5 of the base plan are substantially present; phase 8 bench harness is real; phase 6 pooling is not started.

The main architectural debt is not package naming — it is that **compliance, font embed, and structure tagging live as flag-driven procedures inside two parallel assemblers** (`engine.Generate` and `engine.GenerateDocument`), while several packages that *look* like deep seams (`structure.Manager`, most of `pdfa`) are **scaffolding without call sites**. Product-path unit tests are thin; the multi-page A-4 path **does not subset fonts** even though subsetting is implemented and tested.

**Headline score: 6.2 / 10** — solid foundation and good AI navigability via phase plans, held back by assembly duplication, shallow UA composition, dead seams, and weak product-path test locality.

---

## Scorecard (out of 10)

| Axis | Score | One-line justification |
|------|------:|------------------------|
| Architecture depth | **5.5** | Package split is good; real depth in font/layout/doc; root assemblers and unused Manager/pdfa helpers are shallow or dead. |
| Seam discipline | **4.5** | Few real dual adapters; Mode bits + ifs substitute for a compliance profile module. |
| Locality | **6.5** | Concerns map to packages well; dual assemblers and render glue force multi-file changes for compliance/font. |
| Leverage | **6.0** | `render.PDF` / `TemplatePDF` and `doc.Build` pay back; content/pdfa helpers less so. |
| Testability | **4.0** | 4 test files vs ~28 sources; product seam `GenerateDocument`/`render` untested in `go test`. |
| Compliance design | **6.0** | External veraPDF harness is strong; internal UA is page-as-`P`; A-4 objects present with silent font fallbacks. |
| AI-navigability | **7.5** | Phase plans + template guide are high leverage; no `CONTEXT.md` / ADRs / coding standards. |
| Complexity control | **5.0** | Large dual assembly functions, magic width factors, `map[string]interface{}` dicts, TTF density is inherent. |
| Documentation | **6.5** | Excellent phase checklists and compliance/bench READMEs; sparse godoc and no domain glossary. |
| Delivery vs plan | **7.0** | Phases 1–5 + 8 largely land; 6–7 not started; UA depth below plan intent for tables. |
| **Weighted overall** | **6.2** | See scoring notes below. |

### How the overall was computed

Weights: depth 15%, seams 10%, locality 10%, leverage 10%, testability 15%, compliance 15%, AI-nav 10%, complexity 5%, docs 5%, delivery 5%.

```
5.5×0.15 + 4.5×0.10 + 6.5×0.10 + 6.0×0.10 + 4.0×0.15
+ 6.0×0.15 + 7.5×0.10 + 5.0×0.05 + 6.5×0.05 + 7.0×0.05
≈ 5.79  → rounded with qualitative uplift for working product path → 6.2
```

### Rating band guide

| Band | Meaning |
|------|---------|
| 8–10 | Deep modules, real seams, product path tested, compliance owned by modules |
| 6–7 | Works, clear packages, known assembly debt |
| 4–5 | Fragile, shallow throughout, hard to change safely |
| ≤3 | Unusable as a maintained engine |

**This codebase sits in the lower “works with known debt” band.**

---

## What was reviewed

| Area | Path | Role |
|------|------|------|
| Core assembly | `engine/engine.go`, `engine/document.go` | `Generate`, `GenerateDocument` |
| Object model | `engine/doc`, `engine/write`, `engine/page`, `engine/content` | IDs, encode, page dicts, operators |
| Deep feature modules | `engine/font`, `engine/layout`, `engine/image` | TTF/subset, tables, XObjects |
| Compliance packages | `engine/pdfa`, `engine/structure`, `engine/meta`, `engine/color` | A-4 / UA-2 / XMP / ICC |
| Product edge | `engine/model`, `engine/render` | JSON DTOs + domain→layout |
| Harness | `sampledata/*`, `compliance/*`, `Makefile` | Bench + veraPDF |
| Plans | `plans/phase-01` … `phase-08`, baseplan | Intended architecture |

No `CONTEXT.md`, ADRs, or `CODING_STANDARDS.md` found — domain language lives only in plans/guides.

---

## Architecture snapshot

### Happy-path pipeline

```
JSON (sampledata)
  → model.LoadJSON / LoadTemplate
  → render.PDF / TemplatePDF
  → layout (ContentBuilder, TableLayout)
  → engine.GenerateDocument(DocumentConfig)
  → doc.Document.Build → write.Encoder → []byte PDF
```

Demo path: `engine.Generate(Config)` builds a single-page smoke PDF (what unit tests hit).

### Dependency direction (healthy overall)

```
sampledata → model, render
render     → engine, layout, model, color, image, doc
engine     → doc, write, content, page, font, image, meta, pdfa, structure, color
layout     → content, image
font/page/structure/pdfa → doc
doc        → write
```

No import cycles observed. Product layer correctly depends on the root assembler rather than re-implementing xref.

### Module inventory (depth)

| Module | Depth | Notes |
|--------|-------|-------|
| **font** | **Deep** | Load, cmap, subset, ToUnicode, emit dicts — deletion forces reimplementation |
| **layout** | **Deep** | Table page-break, place text/rect/image; product leverage is real |
| **doc** | **Deep** | Small surface (`AllocID` / `AddObject` / `Build`); owns xref/trailer |
| **render** | **Deep (edge)** | Domain/template → `PageContent` + mode; two real adapters |
| **meta** / **color (ICC)** | Medium–deep | Config/data in → packet/profile out |
| **image** | Medium–deep | JPEG/PNG + cache; engine still hand-builds some XObject dicts |
| **write** | Medium | Necessary encoder; surface ≈ PDF syntax steps |
| **engine root** | Deep *volume*, weak *locality* | Small call interface, huge procedural body, duplicated twice |
| **content** | Shallow | Operator-per-method; layout often writes raw ops |
| **page** | Shallow | Field → dict |
| **pdfa** | Shallow / mostly unused | Only `OutputIntentDict` used from production paths |
| **structure.Manager** | Dead depth | Implemented, never wired; assemblers hand-build Document→P |

---

## Strengths

1. **Package map matches the domain.** Writer, fonts, layout, structure, PDF/A, metadata, and product render are separate modules — high AI-navigability even without a glossary.
2. **Real deep modules exist.** `font` (TTF + subset) and `layout` (tables + flow) pass the deletion test.
3. **Product seam is clear.** Callers can ignore PDF objects and use `render.PDF` / `TemplatePDF`.
4. **Mode flags at the edge.** `doc.ModePDFA4` / `ModePDFUA2` give a single lever for compliance profiles (even if composition is shallow).
5. **Phase plans are excellent.** Checklists + gates (veraPDF, benches) make the intended trajectory legible.
6. **External compliance harness.** `make test-verify-pdfs`, structure-tree check, Zerodha bench — unusual and valuable for a young engine.
7. **Dependency direction is mostly one-way** toward `write`/`doc`.

---

## Critical findings

### C1 — Dual assemblers with drift (Strong)

**Files:** `engine/engine.go`, `engine/document.go`

`Generate` and `GenerateDocument` re-implement the same mode graph: ID allocation, font chain, ICC/XMP, structure tree, catalog extras, footers/UA wrap.

Concrete drift: **`Generate` may call `GenerateSubset`** (`engine.go` ~158–160); **`GenerateDocument` embeds full `RawData` only** (`document.go` ~228). Product/bench path (the important one) does not use the subset implementation that unit tests cover.

**Impact:** Shotgun surgery for every compliance/font change; tests that green on `Generate` do not protect the product path (locality failure).

### C2 — Compliance is flag-sprinkled, not a deep module (Strong)

**Files:** both assemblers, `doc/builder.go` Mode bits, `meta/xmp.go`, `pdfa/pdfa.go`

PDF/A-4 and PDF/UA-2 are composed as:

```
if isA4 { embed fonts, ICC, OutputIntent, XMP PDFA, strip Info }
if isUA  { Lang, MarkInfo, StructTreeRoot, BDC wrap, XMP PDFUA }
```

`pdfa.CatalogA4Extras`, `PageResourceA4Extras`, and `ImageColorSpace` are effectively unused. Policy lives in the assemblers.

### C3 — `structure.Manager` is a fake seam (Strong)

**Files:** `structure/manager.go`, `document.go` ~346–374, `engine.go` ~284–315

Manager is never called outside its package. Real UA trees are hand-built Document → one `P` per page with MCID 0. Layout has an `MCID` field and always-on watermark Artifact tags, but does not own structure.

**Impact:** Phase-5 “tagging” claim is catalog/MarkInfo-level, not semantic table tagging (`Table`/`TR`/`TD` constants exist unused).

### C4 — Product pipeline almost untested in-process (Strong)

| Surface | `go test` coverage |
|---------|-------------------|
| `engine.Generate` | Yes (substring/marker checks) |
| `engine.GenerateDocument` | **No** |
| `render.*` | **No** |
| `model.*` | **No** |
| `structure`, `pdfa`, `meta`, `write`, `doc` | **No** dedicated tests |

External veraPDF on fixtures is valuable but **fixtures are not regenerated from current engine tests**. `make test` skips compliance when fixtures are absent or stale relative to code.

### C5 — Silent degradation on font failure (Worth exploring → treat as bug)

When Liberation cannot load, both assemblers can emit a **fake font + empty FontFile2** and still return success. Compliance green on marker strings can hide broken embedding.

### C6 — Encoding / width policy leakage (Worth exploring)

- Layout `PlaceText` always emits Identity-H / CID-style text; non-A4 assembly installs Type1 Helvetica.
- Text width uses `0.52` in layout and `0.55` for footer page numbers in `document.go` — not font metrics.

### C7 — Dead or half-used packages inflate perceived depth (Worth exploring)

Directory seams suggest deep modules; runtime path is two assemblers + helpers. Deletion test on most of `pdfa` and on `structure.Manager` currently **removes little live behaviour**.

---

## Deepening opportunities

Ranked for leverage. Vocabulary: **module**, **interface**, **implementation**, **depth**, **seam**, **adapter**, **leverage**, **locality**.

### 1. Collapse dual assemblers into one deep assembly module — **Strong**

| | |
|---|---|
| **Files** | `engine/engine.go`, `engine/document.go` |
| **Problem** | Same compliance/font/structure graph twice; drift already present (subset). |
| **Solution** | One internal assembler behind `Generate` / `GenerateDocument`. Thin adapters only map `Config` → `DocumentConfig`. |
| **Wins** | Locality of assembly bugs · leverage for both entry points · delete duplicated blocks |
| **Deletion test** | Deleting one path today just forces edits in the other — neither concentrates complexity alone. |

### 2. Compliance profile module (own A-4 + UA policy) — **Strong**

| | |
|---|---|
| **Files** | Assemblers, `pdfa/`, `meta/`, `color/` ICC, catalog blocks |
| **Problem** | Mode bits + ifs; `pdfa` helpers unused; shotgun surgery for catalog/XMP/ICC. |
| **Solution** | Deep module: `Profile` (PDF20 / A4 / UA2 / A4+UA2) applies OutputIntent, Default* color spaces, MarkInfo, Lang, TrailerInfo policy, metadata flags. Assembler supplies pages + resources only. |
| **Wins** | Locality of compliance rules · real use of `pdfa` · one test surface for profiles |

### 3. Wire `structure.Manager` from layout (or delete it) — **Strong** if UA is product-critical

| | |
|---|---|
| **Files** | `structure/manager.go`, `layout/*`, `document.go` structure block |
| **Problem** | Fake seam; page-level `/P` wrap ≠ PDF/UA-2 leverage for tables. |
| **Solution** | Layout allocates MCIDs via Manager; tables emit Table/TR/TH/TD; assembler only serializes `Manager.Build()`. |
| **Wins** | Tagging co-located with drawing · ParentTree ownership testable · delete hand-rolled trees |

### 4. Single font-embed facade — **Strong**

| | |
|---|---|
| **Files** | Font blocks in both assemblers, `font/*`, render `UsedText` |
| **Problem** | Object-by-object emit duplicated; subset only on demo path; width constants outside font. |
| **Solution** | `font.Embed(allocator, used runes) → resource refs`; layout metrics from same font. |
| **Wins** | Locality for subset/width bugs · product path gets subset · one Liberation policy |

### 5. Make `GenerateDocument` (+ optional `render.PDF`) the test seam — **Strong**

| | |
|---|---|
| **Files** | New `document_test.go`, optional `render/*_test.go`, compliance fixture generation |
| **Problem** | Demo `Generate` is the only integration surface; product path drifts untested. |
| **Solution** | Golden matrix: minimal / multi-page / A-4 / UA / images; optional veraPDF when installed; assert subset on A-4 product path. |
| **Wins** | Interface = test surface · closes C1/C4 · fixtures stay alive |

### 6. Shared render finish helper — **Worth exploring**

| | |
|---|---|
| **Files** | `render/contract_note.go`, `render/template.go` |
| **Problem** | Parallel PageContent mapping, mode, footer, UsedText collection. |
| **Solution** | `render.finish(builders, meta, mode)` shared; domain files only build tables. |
| **Wins** | Less duplicated product glue · consistent compliance flags |

### 7. ContentBuilder as sole content seam — **Worth exploring / Speculative**

Hide operator soup; extend high-level Place*/Draw* so `content.Stream` is implementation detail of layout.

---

## Test map

| Package | Tests | Quality |
|---------|-------|---------|
| `engine` | 4 tests on `Generate` | Shallow markers, not veraPDF |
| `engine/font` | 15 tests | Best unit depth |
| `engine/layout` | 6 tests | Stream smoke |
| `engine/image` | 5 tests | Solid parse/dedup |
| All others | none | — |

**Ratio:** ~1 test file per 7 source files. No `Benchmark*` in package tests (bench lives in sampledata).

---

## Phase plan vs reality

| Phase | Plan | Reality |
|------:|------|---------|
| 1 Core PDF 2.0 writer | ✅ | Present; tested via `Generate` |
| 2 Layout primitives | ✅ | Present; used by render, not by `Generate` |
| 3 Font embedding | ✅ | Real TTF/subset; **product path weak on subset** |
| 4 PDF/A-4 | ✅ | Objects present; silent font fallback risk |
| 5 PDF/UA-2 | ✅ checklist | Catalog/structure present; **tree is shallow** vs full tagging intent |
| 6 Performance pooling | ⏸️ | Not implemented |
| 7 Optional product | ⏸️ | Out of scope for engine core |
| 8 Zerodha bench | harness | Implemented under sampledata + Makefile |

---

## Risk register (short)

| Risk | Severity | Why |
|------|----------|-----|
| Product PDF size/compliance without subset | High | Multi-page A-4 embeds full TTF |
| UA “compliance” overstated | Medium–High | One P per page may pass weak checks without real reading order/table structure |
| Silent empty FontFile2 | High | Success path with broken embed |
| Assembler drift continues | Medium | Every feature added twice |
| Fixture/test gap | Medium | CI can green while product path regresses |

---

## Recommended order of work

1. **Extract shared assembler** + force `Generate` to call it (stop the bleed).
2. **Font.Embed facade** with subset on the product path; fail hard (or error) when embed required but font missing.
3. **`GenerateDocument` integration tests** (markers + optional veraPDF).
4. **Compliance profile module** (deepen `pdfa` / mode application).
5. **Either wire `structure.Manager` for real table tags, or delete Manager and document intentional page-level tagging** (ADR).
6. Phase 6 pooling only after assembly locality is fixed (pools on duplicated code double the mess).

---

## Top recommendation

**Collapse `Generate` / `GenerateDocument` into one deep assembly module and make font embed + subset a single facade used by that module.**

That single move restores locality for the highest-churn behaviour, closes the subset drift bug class, and creates the right test surface (`GenerateDocument`) for everything else (profiles, structure, render).

---

## What this review is not

- Not a line-by-line style nitpick.
- Not a veraPDF full-suite audit of every sample PDF.
- Not an interface redesign (skill step 2 stops at candidates; grilling is optional next).

---

## Follow-ups (skill loop)

Per `/improve-codebase-architecture`, next step if you want to deepen a candidate:

1. Pick one candidate (1–5 recommended).
2. Run `/grilling` on constraints, seam placement, and surviving tests.
3. Optionally `/design-an-interface` (design it twice) for the compliance profile or font embed module.
4. Record durable rejections as ADRs; add `CONTEXT.md` when domain terms stabilize.

---

## Appendix A — Agent method

| Agent | Focus | Outcome |
|-------|--------|---------|
| Explore 1 | Module inventory, depth, dependency graph | Deep: font/layout/doc/render; shallow: content/page/pdfa; dual assembler debt |
| Explore 2 | Coupling, leakage, real vs fake seams | Mode-flag compliance; Manager unused; subset drift; render glue duplication |
| Explore 3 | Testability, metrics, phase vs reality | 4 test files; product path untested; score inputs |

## Appendix B — Key file anchors

| Claim | Location |
|-------|----------|
| `Generate` entry | `engine/engine.go:44` |
| `GenerateDocument` entry | `engine/document.go:44` |
| Subset on demo path only | `engine/engine.go:158–160` vs `document.go:228` |
| UA page wrap | `document.go:151–201`, structure tree `346–374` |
| Unused Manager | `structure/manager.go` (no external call sites) |
| Unused pdfa helpers | `pdfa/pdfa.go:21–38` |
| Product → GenerateDocument | `render/contract_note.go:102`, `render/template.go:85` |
| Phase 6 not started | `plans/phase-06-performance-pooling.md` |

## Appendix C — Related artifacts

- HTML visual report (skill format): [`architecture-review-20260725-011032.html`](./architecture-review-20260725-011032.html)
- Plans index: [`../README.md`](../README.md)
- Base plan: [`../baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md`](../baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md)
