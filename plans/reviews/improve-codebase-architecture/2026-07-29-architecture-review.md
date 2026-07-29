# Architecture Review — gocorepdfengine (re-review)

| | |
|---|---|
| **Date** | 2026-07-29 |
| **Method** | `/improve-codebase-architecture` + `/codebase-design` vocabulary; delta re-score vs 2026-07-25 |
| **Agents** | This re-review (Grok Build): full pass over `engine/**`, harness, tooling, and prior C1–C7 / candidates 1–7 |
| **Scope** | `engine/**`, `sampledata/**`, `plans/**`, `compliance/**`, `Makefile`, `.golangci.yml`, `codehound.toml` |
| **Overall rating** | **6.4 / 10** (was **6.2** on 2026-07-25; **Δ +0.2**) |
| **Tool** | Grok Build |
| **Branch** | `base/b4-codehound-allints` |

---

## Executive summary

Since 2026-07-25 the codebase gained **real operational hygiene** (`.golangci.yml`, `make lint` / `lint-all`, CodeHound config) and **measured performance work** (bench baselines, claimed ~8–12% then +0.66% on compliance-related runs). Assemblers were **internally refactored** into named helpers (`addFontChain`, `addStructureTree`, `buildA4FontChain`, shared `createCatalogDict`), which improves readability and cyclo-facing complexity control.

**Architectural debt is largely unchanged.** Dual assemblers (`Generate` / `GenerateDocument`) still re-implement the mode graph. Product path **`GenerateDocument` still embeds full `RawData` and does not call `GenerateSubset`**. `structure.Manager` is still unwired. Most of `pdfa` remains unused beyond `OutputIntentDict`. Unit tests are still **4 `*_test.go` files**; `GenerateDocument`, `render`, and `model` have no in-process tests.

**Headline score: 6.4 / 10** — modest uplift for lint tooling, helper extraction, and harness maturity; not a structural re-architecture. Still in the lower “works with known debt” band.

---

## Scorecard (out of 10)

| Axis | 2026-07-25 | 2026-07-29 | Δ | One-line justification |
|------|----------:|----------:|--:|------------------------|
| Architecture depth | **5.5** | **5.7** | **+0.2** | Helper extraction inside assemblers; Manager/pdfa still dead depth. |
| Seam discipline | **4.5** | **4.7** | **+0.2** | Shared `createCatalogDict`; Mode bits + ifs still the compliance seam. |
| Locality | **6.5** | **6.7** | **+0.2** | Font/structure helpers local to `document.go`; dual paths still force dual edits. |
| Leverage | **6.0** | **6.1** | **+0.1** | Same `render` / layout leverage; tiny catalog/compress sharing. |
| Testability | **4.0** | **4.0** | **0** | Still 4 test files; product seam untested in `go test`. |
| Compliance design | **6.0** | **6.1** | **+0.1** | Harness + minor fixture/compliance fixes; UA still page-as-`P`. |
| AI-navigability | **7.5** | **7.8** | **+0.3** | Lint + CodeHound + phase plans; still no `CONTEXT.md` / ADRs. |
| Complexity control | **5.0** | **5.6** | **+0.6** | golangci-driven cleanups; functions split; dual graph remains. |
| Documentation | **6.5** | **6.6** | **+0.1** | Tooling configs as operational docs; godoc/glossary still thin. |
| Delivery vs plan | **7.0** | **7.3** | **+0.3** | Lint gate + bench baselines; phases 6–7 still open; UA depth still shallow. |
| **Weighted overall** | **6.2** | **6.4** | **+0.2** | See scoring notes below. |

### Score delta table (compact)

| Axis | Old → New | Δ |
|------|-----------|:-:|
| Architecture depth | 5.5 → 5.7 | +0.2 |
| Seam discipline | 4.5 → 4.7 | +0.2 |
| Locality | 6.5 → 6.7 | +0.2 |
| Leverage | 6.0 → 6.1 | +0.1 |
| Testability | 4.0 → 4.0 | 0 |
| Compliance design | 6.0 → 6.1 | +0.1 |
| AI-navigability | 7.5 → 7.8 | +0.3 |
| Complexity control | 5.0 → 5.6 | +0.6 |
| Documentation | 6.5 → 6.6 | +0.1 |
| Delivery vs plan | 7.0 → 7.3 | +0.3 |
| **Overall** | **6.2 → 6.4** | **+0.2** |

### How the overall was computed

Weights (unchanged): depth 15%, seams 10%, locality 10%, leverage 10%, testability 15%, compliance 15%, AI-nav 10%, complexity 5%, docs 5%, delivery 5%.

```
5.7×0.15 + 4.7×0.10 + 6.7×0.10 + 6.1×0.10 + 4.0×0.15
+ 6.1×0.15 + 7.8×0.10 + 5.6×0.05 + 6.6×0.05 + 7.3×0.05
= 0.855 + 0.470 + 0.670 + 0.610 + 0.600
+ 0.915 + 0.780 + 0.280 + 0.330 + 0.365
= 5.875  → qualitative uplift for working product path + lint/bench maturity → 6.4
```

Prior review raw was ≈5.79 with uplift → 6.2. Raw rose ~0.09 from real hygiene/complexity work; overall band +0.2.

### Rating band guide

| Band | Meaning |
|------|---------|
| 8–10 | Deep modules, real seams, product path tested, compliance owned by modules |
| 6–7 | Works, clear packages, known assembly debt |
| 4–5 | Fragile, shallow throughout, hard to change safely |
| ≤3 | Unusable as a maintained engine |

**This codebase remains in the lower “works with known debt” band** (closer to mid-band than before only by tooling maturity, not by seam redesign).

---

## What was reviewed

| Area | Path | Role |
|------|------|------|
| Core assembly | `engine/engine.go`, `engine/document.go` | `Generate`, `GenerateDocument` + helpers |
| Object model | `engine/doc`, `engine/write`, `engine/page`, `engine/content` | IDs, encode, page dicts, operators |
| Deep feature modules | `engine/font`, `engine/layout`, `engine/image` | TTF/subset, tables, XObjects |
| Compliance packages | `engine/pdfa`, `engine/structure`, `engine/meta`, `engine/color` | A-4 / UA-2 / XMP / ICC |
| Product edge | `engine/model`, `engine/render` | JSON DTOs + domain→layout |
| Harness | `sampledata/*`, `compliance/*`, `Makefile`, `baselines/*` | Bench + veraPDF |
| Tooling (new since last review) | `.golangci.yml`, `codehound.toml`, `make lint` | Static analysis gates |
| Plans | `plans/phase-01` … `phase-08`, baseplan | Intended architecture |
| Prior review | `2026-07-25-architecture-review.md` | Baseline scores + C1–C7 |

No `CONTEXT.md`, ADRs, or `CODING_STANDARDS.md` found.

---

## Architecture snapshot

### Happy-path pipeline (unchanged)

```
JSON (sampledata)
  → model.LoadJSON / LoadTemplate
  → render.PDF / TemplatePDF
  → layout (ContentBuilder, TableLayout)
  → engine.GenerateDocument(DocumentConfig)
  → doc.Document.Build → write.Encoder → []byte PDF
```

Demo path: `engine.Generate(Config)` — single-page smoke PDF (what unit tests hit).

### Current shape (reality 2026-07-29)

```
sampledata → model, render
render     → engine.GenerateDocument, layout, model, color, image
GenerateDocument / Generate  (dual procedural assemblers)
  → shared createCatalogDict (small win)
  → font embed: Generate may subset; GenerateDocument uses RawData only
  → structure dicts hand-built Document→P; Manager unused
  → pdfa.OutputIntentDict only; CatalogA4Extras / PageResourceA4Extras unused
  → doc.Build → write
```

### Dependency direction (healthy overall — unchanged)

```
sampledata → model, render
render     → engine, layout, model, color, image, doc
engine     → doc, write, content, page, font, image, meta, pdfa, structure, color
layout     → content, image
font/page/structure/pdfa → doc
doc        → write
```

No import cycles observed.

### Module inventory (depth)

| Module | Depth | Notes vs last review |
|--------|-------|----------------------|
| **font** | **Deep** | Subset implementation still solid; product integration still skips subset |
| **layout** | **Deep** | Tables + flow; watermark Artifact test strengthened |
| **doc** | **Deep** | AllocID / AddObject / Build |
| **render** | **Deep (edge)** | Still two parallel finish paths to GenerateDocument |
| **meta** / **color (ICC)** | Medium–deep | Unchanged role |
| **image** | Medium–deep | Unchanged |
| **write** | Medium | Unchanged |
| **engine root** | Deep volume, weak locality | Helpers extracted; **two** assembly graphs remain |
| **content** | Shallow | Unchanged |
| **page** | Shallow | Unchanged |
| **pdfa** | Shallow / mostly unused | Still only `OutputIntentDict` on live path |
| **structure.Manager** | Dead depth | Still no external call sites |

---

## Strengths

1. **Package map still matches the domain** — high AI-navigability.
2. **Real deep modules** — `font` (TTF + subset) and `layout` (tables) pass the deletion test.
3. **Product seam remains clear** — `render.PDF` / `TemplatePDF` → `GenerateDocument`.
4. **Phase plans remain excellent** checklists + gates.
5. **External compliance + bench harness matured** — x10 baselines under `baselines/`, Makefile targets expanded.
6. **NEW: lint / CodeHound tooling** — `.golangci.yml`, `make lint`, `make lint-all`, `codehound.toml` raise the floor for change safety.
7. **NEW: assembler internal structure** — `addFontChain`, `addLoadedFontChain`, `addStructureTree`, `buildA4FontChain`, shared `createCatalogDict` reduce monolith opacity *within* each path.
8. **Dependency direction still one-way** toward `write`/`doc`.

---

## Critical findings (status vs 2026-07-25)

### C1 — Dual assemblers with drift — **STILL OPEN** (partial internal cleanup)

**Status:** STILL OPEN · partial: helpers extracted; catalog dict shared  
**Files:** `engine/engine.go`, `engine/document.go`

| Entry | Location | Subset? |
|-------|----------|---------|
| `Generate` | `engine/engine.go:124` | **Yes** via `GenerateSubset` (`engine.go:46–48`) |
| `GenerateDocument` | `engine/document.go:48` | **No** — `compressData(loadedFont.RawData)` (`document.go:364`) |

Shared progress: both call `createCatalogDict` (`engine.go:318`, used from `engine.go:280` and `document.go:223`). Font/structure/XMP/ICC graphs remain duplicated as separate helper trees (`buildA4FontChain` vs `addLoadedFontChain` / `addFallbackFontChain`).

**Impact unchanged:** shotgun surgery; tests on `Generate` do not protect product path.

### C2 — Compliance is flag-sprinkled, not a deep module — **STILL OPEN**

**Status:** STILL OPEN  
**Evidence:** `isA4` / `isUA` mode forks in both assemblers (`document.go:69–74`, `engine.go:130–135`); `pdfa.CatalogA4Extras`, `PageResourceA4Extras`, `ImageColorSpace` defined in `pdfa/pdfa.go:21–38` with **no callers outside `pdfa`**. Live use remains `pdfa.OutputIntentDict` only (`document.go:199`, `engine.go:256`).

### C3 — `structure.Manager` is a fake seam — **STILL OPEN**

**Status:** STILL OPEN  
**Evidence:** `NewManager` only at `structure/manager.go:16`. Grep finds **zero** call sites outside that package. Live UA trees still hand-built Document → one `P` per page, MCID 0 (`document.go:309–337`, `engine.go:285–316`). Table struct types (`STable`/`STR`/`STH`/`STD` in `structure.go`) unused by layout.

### C4 — Product pipeline almost untested in-process — **STILL OPEN**

**Status:** STILL OPEN  

| Surface | `go test` coverage (2026-07-29) |
|---------|--------------------------------|
| `engine.Generate` | Yes (4 tests in `engine_test.go`) |
| `engine.GenerateDocument` | **No** |
| `render.*` | **No** |
| `model.*` | **No** |
| `structure`, `pdfa`, `meta`, `write`, `doc` | **No** dedicated tests |

Still **4 test files** vs **28 source files** under `engine/`. External veraPDF fixtures remain valuable but not regenerated from product-path unit tests.

### C5 — Silent degradation on font failure — **STILL OPEN**

**Status:** STILL OPEN  
**Evidence:** `addFallbackFontChain` (`document.go:385–396`) and fake-font branch in `buildA4FontChain` (`engine.go:79–101`) still emit empty FontFile2 (`"/Length": 0`) and return success.

### C6 — Encoding / width policy leakage — **STILL OPEN**

**Status:** STILL OPEN  
**Evidence:**
- Layout width factor `0.52` — `layout/layout.go:63`
- Footer page-number width factor `0.55` — `document.go:256`
- Non-A4 Type1 Helvetica vs Identity-H/CID text on A-4 paths unchanged

### C7 — Dead or half-used packages inflate perceived depth — **STILL OPEN**

**Status:** STILL OPEN  
**Evidence:** Deletion of `structure.Manager` and most of `pdfa` (except `OutputIntentDict`) would remove almost no live product behaviour. Directory seams still oversell runtime depth.

---

## Progress since last review (2026-07-25 → 2026-07-29)

| Area | What improved | Architectural effect |
|------|---------------|----------------------|
| Lint gate | `.golangci.yml` + `make lint` / `lint-all` | Complexity control, delivery hygiene; not a seam redesign |
| CodeHound | `codehound.toml` + local binary | AI/tooling navigability for static findings |
| Assembler readability | Extracted helpers; shared `createCatalogDict` | Locality *within* files; dual policy graph remains |
| golangci cleanups | Broad edits across color, font, layout, structure, render, model | Style/complexity; subset drift **not** fixed |
| Benchmarks | `baselines/zerodha_bench_x10*`, bench script updates | Delivery/performance evidence (~8–12%, later +0.66% claims in git history) |
| Layout tests | Watermark CID + Artifact assertions (`layout_test.go`) | Small quality win; not product-path coverage |
| Branch | `base/b4-codehound-allints` | Tooling-first cleanup branch |

**What did *not* move:** dual assembly collapse, product-path subset, Manager wiring, compliance profile module, `GenerateDocument` tests, phase 6 pooling, ADRs/`CONTEXT.md`.

**Note:** `Makefile` around the lint targets contains corrupted comment text (PR UI paste residue near `lint-all`). Cosmetic delivery noise; fix when next editing the Makefile.

---

## Deepening opportunities (updated)

Ranked for leverage. Vocabulary: **module**, **interface**, **implementation**, **depth**, **seam**, **adapter**, **leverage**, **locality**.

### 1. Collapse dual assemblers into one deep assembly module — **Strong** · STILL OPEN

| | |
|---|---|
| **Status** | STILL OPEN (PARTIAL: helpers + shared catalog) |
| **Files** | `engine/engine.go`, `engine/document.go` |
| **Problem** | Same compliance/font/structure graph twice; subset drift still live. |
| **Solution** | One internal assembler behind thin `Generate` / `GenerateDocument` adapters. |
| **Wins** | Locality · leverage · delete duplicated blocks |

### 2. Compliance profile module (own A-4 + UA policy) — **Strong** · STILL OPEN

| | |
|---|---|
| **Status** | STILL OPEN |
| **Files** | Assemblers, `pdfa/`, `meta/`, `color/` ICC |
| **Problem** | Mode bits + ifs; unused pdfa helpers. |
| **Solution** | Deep `Profile` applies OutputIntent, Default* CS, MarkInfo, Lang, TrailerInfo, metadata flags. |

### 3. Wire `structure.Manager` from layout (or delete it) — **Strong** · STILL OPEN

| | |
|---|---|
| **Status** | STILL OPEN |
| **Files** | `structure/manager.go`, `layout/*`, structure blocks in assemblers |
| **Problem** | Fake seam; page-level `/P` ≠ table UA leverage. |
| **Solution** | Layout allocates MCIDs; tables emit Table/TR/TH/TD; or delete Manager + ADR intentional shallowness. |

### 4. Single font-embed facade — **Strong** · STILL OPEN

| | |
|---|---|
| **Status** | STILL OPEN (subset still demo-path only) |
| **Files** | Font helpers in both assemblers, `font/*`, render `UsedText` |
| **Problem** | `UsedText` collected for subsetting comment/field, but product embed uses full `RawData`. |
| **Solution** | `font.Embed(allocator, used runes) → resource refs`; hard error when embed required. |

### 5. Make `GenerateDocument` (+ optional `render.PDF`) the test seam — **Strong** · STILL OPEN

| | |
|---|---|
| **Status** | STILL OPEN |
| **Files** | Need `document_test.go`, optional `render/*_test.go` |
| **Problem** | Interface under test ≠ product interface. |
| **Solution** | Golden matrix: minimal / multi-page / A-4 / UA / images; assert subset on A-4 product path. |

### 6. Shared render finish helper — **Worth exploring** · STILL OPEN

| | |
|---|---|
| **Status** | STILL OPEN |
| **Files** | `render/contract_note.go`, `render/template.go` |
| **Problem** | Parallel PageContent mapping, mode, footer, UsedText. |
| **Solution** | `render.finish(builders, meta, mode)`. |

### 7. ContentBuilder as sole content seam — **Worth exploring / Speculative** · STILL OPEN

Hide operator soup; extend Place*/Draw* so `content.Stream` is layout-internal.

---

## Test map

| Package | Tests | Quality |
|---------|-------|---------|
| `engine` | 4 tests on `Generate` | Shallow markers, not veraPDF; no `GenerateDocument` |
| `engine/font` | 16 tests | Best unit depth (subset included) |
| `engine/layout` | 7 tests | Stream smoke + watermark CID/Artifact |
| `engine/image` | 5 tests | Solid parse/dedup |
| All others | none | — |

**Ratio:** ~1 test file per 7 source files (unchanged). No package-level `Benchmark*`; bench lives in `sampledata` + `baselines/`.

`go test ./...` (2026-07-29): `engine`, `font`, `image`, `layout` ok; remaining packages report no test files.

---

## Phase plan vs reality

| Phase | Plan | Reality (2026-07-29) |
|------:|------|----------------------|
| 1 Core PDF 2.0 writer | ✅ | Present; tested via `Generate` |
| 2 Layout primitives | ✅ | Present; used by render |
| 3 Font embedding | ✅ | Real TTF/subset; **product path still full RawData** |
| 4 PDF/A-4 | ✅ | Objects present; silent font fallback risk |
| 5 PDF/UA-2 | ✅ checklist | Catalog/structure present; **tree still shallow** |
| 6 Performance pooling | ⏸️ | Still not implemented (`sync.Pool` absent in engine) |
| 7 Optional product | ⏸️ | Out of scope for engine core |
| 8 Zerodha bench | harness | Matured baselines + scripts |

**Adjacent delivery (not a plan phase):** lint + CodeHound + assembler micro-refactors + measured throughput work.

---

## Risk register (short)

| Risk | Severity | Why | Trend |
|------|----------|-----|-------|
| Product PDF size/compliance without subset | High | Multi-page A-4 embeds full TTF (`document.go:364`) | Unchanged |
| Silent empty FontFile2 | High | Fallback still success path | Unchanged |
| UA “compliance” overstated | Medium–High | One P per page | Unchanged |
| Assembler drift continues | Medium | Two graphs; only catalog shared | Slightly better (helpers) |
| Fixture/test gap | Medium | CI can green while product path regresses | Unchanged |
| Makefile comment corruption | Low | Paste residue near lint targets | New cosmetic |

---

## Recommended order of work

1. **Extract shared assembler** + force both entry points through it (stop the bleed). *Partial helper work already done — finish the collapse.*
2. **Font.Embed facade** with subset on the product path; fail hard when embed required but font missing.
3. **`GenerateDocument` integration tests** (markers + optional veraPDF; assert subset).
4. **Compliance profile module** (deepen `pdfa` / mode application).
5. **Wire or delete `structure.Manager`** (ADR if intentional page-level tagging).
6. Phase 6 pooling only after assembly locality is fixed.
7. Fix Makefile lint comment noise; keep `make lint` green as a gate.

---

## Top recommendation

**Still the same as 2026-07-25:** collapse `Generate` / `GenerateDocument` into one deep assembly module and make font embed + subset a single facade used by that module.

Lint and helper extraction made that refactor *safer to attempt*; they did not replace it. Closing subset drift on the product path is the highest-leverage correctness + size win available without expanding UA semantics.

---

## What this review is not

- Not a line-by-line style nitpick (golangci already drove that pass).
- Not a veraPDF full-suite audit of every sample PDF.
- Not an interface redesign (skill stops at candidates; grilling optional next).
- Not a re-score of Go style or Ponytail leanness (those are separate reviews).

---

## Follow-ups (skill loop)

1. Pick candidate 1 (assemblers) or 4 (font.Embed) first — both close C1 subset drift.
2. Run `/grilling` on constraints and surviving tests.
3. Optionally `/design-an-interface` for compliance profile or font embed.
4. Record durable rejections as ADRs; add `CONTEXT.md` when domain terms stabilize.
5. After assembler collapse: generate fixtures from `GenerateDocument` tests so compliance stays alive.

---

## Appendix A — Agent method

| Pass | Focus | Outcome |
|------|--------|---------|
| Prior baseline | Read 2026-07-25 MD + HTML + scoring notes | Anchored weights and C1–C7 |
| Code inventory | `engine/**`, tests, Makefile, tooling | Confirmed 4 test files, dual assemblers, lint stack |
| Finding verification | Grep/read for subset, Manager, pdfa, UsedText | C1–C7 statuses with file:line evidence |
| Delivery check | git log, baselines, phase-06 | Perf/lint progress; pooling still absent |
| Synthesis | Weighted re-score + dual deliverables | Overall 6.4 (+0.2) |

## Appendix B — Key file anchors

| Claim | Location |
|-------|----------|
| `Generate` entry | `engine/engine.go:124` |
| `GenerateDocument` entry | `engine/document.go:48` |
| Subset on demo path | `engine/engine.go:46–48` |
| No subset on product path | `engine/document.go:364` (`RawData` only) |
| Shared catalog builder | `engine/engine.go:318` (call sites `:280`, `document.go:223`) |
| UA page wrap / structure | `document.go:228–277`, `309–337` |
| Unused Manager | `structure/manager.go` (no external call sites) |
| Unused pdfa helpers | `pdfa/pdfa.go:21–38` |
| Fake empty FontFile2 | `document.go:385–396`, `engine.go:79–101` |
| Width constants | `layout/layout.go:63` (0.52), `document.go:256` (0.55) |
| Product → GenerateDocument | `render/contract_note.go:104`, `render/template.go:85` |
| UsedText collected (unused for subset) | `document.go:39`, `render/*` → `addLoadedFontChain` |
| Lint config | `.golangci.yml`, `Makefile` `lint` / `lint-all` |
| CodeHound | `codehound.toml` |
| Phase 6 not started | `plans/phase-06-performance-pooling.md` |

## Appendix C — Related artifacts

- HTML visual report (this re-review): [`architecture-review-20260729.html`](./architecture-review-20260729.html)
- Prior architecture review (2026-07-25): [`2026-07-25-architecture-review.md`](./2026-07-25-architecture-review.md)
- Prior HTML: [`architecture-review-20260725-011032.html`](./architecture-review-20260725-011032.html)
- Reviews summary / scoring context: [`../intial-summary.md`](../intial-summary.md)
- Plans index: [`../../README.md`](../../README.md)
- Base plan: [`../../baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md`](../../baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md)
