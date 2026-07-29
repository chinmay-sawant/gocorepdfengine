# Go Code Style Review — gocorepdfengine (re-review)

| | |
|---|---|
| **Date** | 2026-07-29 |
| **Method** | Global skill `golang-code-style` (samber/cc-skills-golang@golang-code-style v1.2.2) |
| **Agents** | Re-review (measurement-driven; prior 2026-07-25 baseline + evidence on changed files) |
| **Scope** | All `.go` under `engine/` and `sampledata/` (~36 files, 16 packages) |
| **Overall rating** | **6.6 / 10** |
| **Tool** | Grok Build |
| **Prior** | [2026-07-25 review](./2026-07-25-golang-code-style-review.md) — overall **6.2** |

---

## Executive summary

Since 2026-07-25 the repo gained **lint infrastructure** (`.golangci.yml`, `make lint` / `make lint-all`, codehound) and a **golangci-driven refactor** that materially shortened the dual assemblers and partially decomposed the TTF subset path. Style debt is **still concentrated** in the same places the prior review called out: **`StyledCell` (~92 one-line 7-arg call sites)**, **5–7 param layout APIs**, and **`map[string]interface{}` PDF AST**.

**Delta vs prior 6.2:** overall **+0.4 → 6.6**. Control flow and function *length* improved most; function *parameter* shape and call-site wrapping barely moved. Values/philosophy remain the strongest axis (stdlib-only, no `reflect`).

| Prior priority item | Status | Evidence |
|---------------------|--------|----------|
| Redesign `StyledCell` / theme API | **STILL OPEN** | `StyledCell` still 7 params; **92** call sites in `render/contract_note.go`, all one-line |
| `LayOut` / `LayOutFrom` options struct | **STILL OPEN** | `layOutFrom` still **7** params; signatures still >120 chars |
| Split dual assemblers | **PARTIAL** | `Generate` ~**161** (was ~300); `GenerateDocument` ~**180** (was ~355); helpers extracted (`buildA4FontChain`, `addFontChain`, `createCatalogDict`, structure emitters). Dual paths remain; new helpers have **8–10** params |
| Split `buildSubsetTTF` | **PARTIAL** | Body ~**183** lines (was ~280); helpers `buildGlyphEntries`, `buildLocaData`, `buildGlyfAndHmtxData`, …; file still **542** lines |
| Typed PDF values + encoding | **PARTIAL** | Catalog/color paths use `write.Ref`; **~26** `fmt.Sprintf("%d 0 R")` remain; **~72** `map[string]interface{}` (was ~77); `%v` default still in encoder |
| Unexport dead surface | **STILL OPEN** | `layout.Point`, `structure.Manager` (+ `NewManager`/`Build`) unused outside package |
| Hygiene (named conditions, `range n`, nil slices, …) | **PARTIAL** | Lint/nolint noise reduced; multi-operand conditions, C-style loops, bare `var s []T` remain |
| Lint tooling | **FIXED** (new) | `.golangci.yml` + `make lint` / `lint-all`; funlen lines:200 |

**Headline score: 6.6 / 10** — still in the “readable with concentrated debt” band, but assembler god-function risk is down and tooling now enforces a floor.

---

## Scorecard (out of 10)

| Axis | Prior | Now | Δ | One-line justification |
|------|------:|----:|--:|------------------------|
| Control flow | 6.5 | **7.0** | +0.5 | Font paths flattened with early returns; remaining multi-operand / C-style loops |
| Function design & variables | 5.5 | **6.0** | +0.5 | Assembler/subset lengths halved; StyledCell + new 8–10 param helpers |
| Line length, breaking & file org | 5.5 | **5.8** | +0.3 | Better file structure; ~92 StyledCell one-liners + mega-literals dominate |
| Values, strings, types & philosophy | 7.4 | **7.5** | +0.1 | Partial `write.Ref` adoption; still `interface{}` AST / unescaped PDF strings |
| **Weighted overall** | **6.2** | **6.6** | **+0.4** | Equal-weight average of the four axes |

### How the overall was computed

Equal weights (25% each):

```
7.0×0.25 + 6.0×0.25 + 5.8×0.25 + 7.5×0.25
= 1.75 + 1.50 + 1.45 + 1.875
= 6.575 → 6.6
```

### Rating band guide

| Band | Meaning |
|------|---------|
| 8–10 | Short focused functions, options structs, early returns, named composites, narrow types |
| 6–7 | Readable overall; known concentrated style debt |
| 4–5 | Widespread nesting, long param lists, unclear organization |
| ≤3 | Unreadable / hostile to maintenance |

**This codebase remains in the lower “readable with concentrated style debt” band**, edging upward from the 2026-07-25 floor.

---

## What was reviewed

| Area | Path | Style focus |
|------|------|-------------|
| Assemblers | `engine/engine.go`, `engine/document.go` | Length, nesting, extracted helpers, param counts |
| Layout / theme | `engine/layout/*` | `StyledCell`, `LayOut*`, call wrapping |
| Font pipeline | `engine/font/*` | Subset split, switch/range, strings |
| Render / model | `engine/render/*`, `engine/model/*` | Cell APIs, positional literals |
| Encode / write | `engine/write/*`, `engine/doc/*` | `interface{}`, `Ref`/`PDFString` |
| Structure / pdfa / meta / color / image / page / content | package files | Conditions, exports, strings |
| Samples | `sampledata/**` | Bench length, range loops |
| Tooling | `.golangci.yml`, `Makefile`, `codehound` | Lint floor (context, not scored as skill axis) |

Skill rules applied (not naming/lint/godoc — those are sibling skills):

- Line length & breaking (~120; 4+ args one-per-line)
- Variable declarations (`:=` vs `var`; non-nil slices/maps; named composite fields)
- Control flow (early return; no useless else; named multi-operand conditions; switch; range)
- Function design (short; ≤4 params; context order; naked returns)
- Value vs pointer args
- Code organization within files
- String handling & type conversions
- Philosophy (deps, reflect, public surface)

### Key measurements (2026-07-29)

| Metric | Prior (~2026-07-25) | Now |
|--------|--------------------:|----:|
| `Generate` body (approx lines) | ~300 | **~161** |
| `GenerateDocument` body | ~355 | **~180** |
| `buildSubsetTTF` body | ~280 | **~183** |
| `runBenchmark` body | ~245 | **~248** |
| Production funcs ≥200 lines | 3 | **0** (bench only ≥200) |
| Lines >120 in `engine/` | ~20 | **26** |
| Lines >120 in `sampledata/` | 0 | **0** |
| `StyledCell` call sites | ~92 | **92** (+ def) |
| Funcs with ≥5 params | (not tallied) | **16** |
| `map[string]interface{}` sites | ~77 | **~72** |
| `fmt.Sprintf("%d 0 R")` | ~30 | **~26** |
| External deps / `reflect` | 0 / 0 | **0 / 0** |
| Blank / dot imports | 0 | **0** |

---

## Dimension reports

### 1. Control flow — **7.0 / 10** (was 6.5)

#### Prior high findings status

| Prior | Status | Notes |
|-------|--------|-------|
| Nested font path in `Generate` | **PARTIAL → improved** | Happy path delegated to `buildA4FontChain`; `Generate` body is flat mode branches + helpers |
| Nested font path in `GenerateDocument` | **FIXED (style)** | `addFontChain` uses early return for non-A4 / load-fail; `addLoadedFontChain` / `addFallbackFontChain` split |
| `buildSubsetTTF` success-if / error-else for head/hhea | **FIXED** | Missing head/hhea now `return nil, errors.New(...)` |

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | medium | `engine.go` `buildA4FontChain` | Reduce nesting | Still `if loadedFont != nil { … } else { fallback }` — invert with early nil branch |
| 2 | medium | `font/metrics.go` `BuildCIDToGIDMap` | Nesting | Nested `SubGIDMap` lookup branches |
| 3 | medium | `layout/table.go` `layOutFrom` | Nesting | Image place under `if err == nil` (prefer continue/early) |
| 4 | medium | `render/template.go` | Nesting | Repeated `if err == nil { … }` color/image blocks |
| 5 | medium | `color/hex.go` `ParseHex` | Named booleans | `err1 \|\| err2 \|\| err3` |
| 6 | medium | `image/image.go` `NewFromJPEG` | Named booleans | Multi-operand SOI check |
| 7 | medium | `model/contract_note.go` `ExpandTrades` | Named booleans | 3-operand retail keep condition |
| 8 | medium | `font/ttf.go` `parseName` | Prefer switch | `platformID` if / else-if |
| 9 | medium | `font/ttf.go` `StemVValue` | Drop else | `else if weight < 700` after returnable branch |
| 10 | medium | subset / model / bench | Prefer `range n` | C-style `for i := 0; i < n; i++` still in several places |
| 11–14 | low | content, write, structure, template | Minor else / nesting | Optional cleanups |

#### Positive patterns

- Early guards common (`structure.Manager` methods, empty text, font load length checks).
- `addFontChain` / `addSinglePageStructureTree` / UA gates use early return well.
- `switch` used well for escapes, object types, alignment, cmap formats, JPEG components.
- `range` over collections is default; some `for range n` / `for range cfg.Pages` adopted.
- Error-first I/O in sample mains remains flat.

#### Stats

| Severity | Count |
|----------|------:|
| high | 0 |
| medium | 10 |
| low | 4 |
| **total** | **14** |

---

### 2. Function design & variable declarations — **6.0 / 10** (was 5.5)

#### Prior high findings status

| Prior | Status | Notes |
|-------|--------|-------|
| `GenerateDocument` ~355 lines | **PARTIAL** | ~180 lines; still multi-concern but under funlen 200 |
| `Generate` ~300 lines | **PARTIAL** | ~161 lines; helpers extracted |
| `buildSubsetTTF` ~280 lines | **PARTIAL** | ~183 lines; table helpers extracted; file 542 LOC |
| `runBenchmark` ~245 lines | **STILL OPEN** | ~248 lines (sampledata; excluded from funlen via config) |

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | **high** | `sampledata/zerodha/bench.go` `runBenchmark` (~248) | Short/focused | Setup + workers + stats + I/O still one body |
| 2 | **high** | `layout/theme.go` `StyledCell` + call graph | ≤4 params | **7** params; ~92 call sites; no options struct migration |
| 3 | medium | `document.go` `GenerateDocument` (~180) | Short/focused | Still owns IDs, pages, meta, structure orchestration |
| 4 | medium | `engine.go` `Generate` (~161) | Short/focused | Improved; still multi-phase assembler |
| 5 | medium | `font/subset.go` `buildSubsetTTF` (~183) | Short/focused | Pack/checksum still inline |
| 6 | medium | Extracted helpers with **8–10** params | ≤4 params | `addStructureTree` (10), `createCatalogDict` (9), `buildA4FontChain` (9), `addSinglePageStructureTree` (8) — extraction without options structs |
| 7 | medium | `layout/table.go` `layOutFrom` / public `LayOut*` | ≤4 params | **5–7** float/page params |
| 8 | medium | `render/template.go` `cellFromProps` (6), `layout.PlaceImage` (6), `content.Tm`/`cm` (6) | ≤4 params | Geometry/matrix APIs |
| 9 | medium | `structure/manager.go` `Build` | Result struct | **7** return values (named results, bare return when disabled is fine; long signature not) |
| 10 | medium | `model/template.go` `PageSizes` | Named fields | Positional `{842, 1191}` etc. |
| 11 | medium | `font/emit.go` `WidthsArray` | Named fields | `cidRange{cid, cid, width}` |
| 12 | medium | Multiple | Non-nil slices/maps | `var ranges []T`, nil `Elements` until first append |
| 13 | medium | `render/contract_note.go` builders | Short/focused | ~80–90 line retail/active/HFT builders (acceptable) |
| 14–16 | low | misc | `var` misuse, matrix APIs | Minor |

`context.Context` is unused in scope — parameter-order rule is N/A.

#### Positive patterns

- Public options structs: `engine.Config`, `engine.DocumentConfig`, `render.Options`.
- Capacity-aware `make` when length is known (`contentIDs`, `pageIDs`, `elemPageIDs`).
- Named fields dominate exported composites.
- TTF parse still split by table (`parseHead`, `parseCMap`, …).
- Contract-note entry stays thin: `PDF` → `BuildTable` → layout → `GenerateDocument`.
- `fontChain` struct groups related object IDs (good partial options-struct pattern).
- Production functions now all **under funlen 200** after the lint pass.

#### Stats

| Severity | Count |
|----------|------:|
| high | 2 |
| medium | 11 |
| low | 3 |
| **total** | **16** |

---

### 3. Line length, breaking & code organization — **5.8 / 10** (was 5.5)

#### Prior high findings status

| Prior | Status | Notes |
|-------|--------|-------|
| ~92 one-line `StyledCell` calls | **STILL OPEN** | Still 92 sites; zero multi-line call sites |
| `LayOut*` signatures >120 | **STILL OPEN** | 123–153 char signatures remain |

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | **high** | `layout/theme.go` + ~92 call sites | 4+ args one-per-line | `StyledCell` always one-line; skill requires one-arg-per-line for 4+ |
| 2 | **high** | `layout/table.go` `LayOut*` | >120 / options struct | 5–7 param signatures exceed ~120 |
| 3 | medium | PlaceImage, cellFromProps, spanProps, Tm/cm, buildGlyphEntries, structure helpers | 4+ args one-per-line | Geometry + new assembler helpers |
| 4 | medium | `engine.go`, `document.go`, `meta/xmp.go` | Break long lines | Multi-hundred / multi-thousand-char ToUnicode / XMP literals |
| 5 | medium | Dual generate paths | Group related logic | Shared `createCatalogDict`; font/XMP still duplicated across paths |
| 6 | medium | Export surface | Unexport aggressively | Dead: `layout.Point`, `structure.Manager`; wide dict helpers |
| 7 | medium | `structure/manager.go` `Build` | Result struct | 7 return values on one long line (~157c) |
| 8–9 | low | multi-type files; string notes | Org / strings | Optional |

**Blank imports:** 0 · **Dot imports:** 0 — compliant.

#### Positive patterns

- Package boundaries still clear (`content`, `doc`, `font`, `layout`, `meta`, `model`, `page`, `pdfa`, `render`, `structure`, `write`).
- Assembler files improved: helpers grouped below `Generate` / `GenerateDocument`.
- Font methods still split by concern across files.
- Most production lines stay ≤120; long-line density moderate and concentrated.
- `sampledata/` has **0** lines >120.
- `CellStyleFromColors` is a small step toward composable cell construction (unused by call sites that still go through `StyledCell`).

#### Stats

| Metric | Value |
|--------|------:|
| Lines >120 (`engine/`) | **26** |
| Lines >120 (`sampledata/`) | **0** |
| `StyledCell` 7-arg one-line calls | **~92** |
| Blank / dot imports | **0** |
| Exported types in `engine/` | **~61** |
| Clearly unused exports | `layout.Point`, `structure.Manager` (+ related) |

---

### 4. Values, strings, types & philosophy — **7.5 / 10** (was 7.4)

#### Prior findings status

| Prior | Status | Notes |
|-------|--------|-------|
| ~77× `map[string]interface{}` | **PARTIAL** | ~72 sites; encoder still falls to `%v` |
| ~30× `fmt.Sprintf("%d 0 R")` | **PARTIAL** | ~26 remain; catalog/ICC paths use `write.Ref` |
| Unescaped PDF strings | **STILL OPEN** | StructElem Title/Alt/Lang; catalog Lang still `"("+lang+")"` |
| Theme colors exported from `color` | **STILL OPEN** | Product palette still in `engine/color` |

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | medium | Cross-cutting PDF dicts | Narrow types / avoid `any` | ~72× `map[string]interface{}`; encoder default `%v` |
| 2 | medium | `font/emit.go` widths / bbox | Explicit encode | `[]int` relies on Go `%v` looking like PDF |
| 3 | medium | page/structure/font/pdfa | strconv / helpers | ~26× `fmt.Sprintf("%d 0 R")` despite `write.Ref` |
| 4 | medium | footer hex, ToUnicode | strconv in loops | `fmt.Sprintf("%04X")` in hot/loop paths |
| 5 | medium | StructElem Title/Alt; catalog Lang | PDF string escape | `fmt.Sprintf("(%s)", …)` / `"("+lang+")"` without `PDFString` |
| 6 | medium | `color` Theme* vars | Minimize public surface | Product palette exported from core `engine/color` |
| 7–10 | low | StringLit buffer, appendStr, exports, DefaultBorder | polish | Incremental |

#### Positive patterns

- **Zero third-party deps** in `go.mod` — still strong “little copying > little dependency.”
- **No `reflect`** in engine/sampledata.
- Small types by value; pointers for mutation, large/`nil`-meaningful cases.
- `strconv` for floats and display ints; `strings.Builder` in footer / stream assembly.
- `%q` common in `color.ParseHex` and many font errors.
- `write.PDFString` used correctly in `pdfa` OutputIntent fields.
- Partial `write.Ref` adoption in catalog and color-space resources.

#### Stats

| Metric | Approx. |
|--------|---------|
| External deps | **0** |
| `reflect` usages | **0** |
| `map[string]interface{}` sites | **~72** |
| `fmt.Sprintf("%d 0 R")` | **~26** |
| Pointless `*string`/`*int` params | **none** |

---

## Priority cleanup order (updated)

| # | Item | Status vs prior | Action |
|---|------|-----------------|--------|
| 1 | Redesign `StyledCell` / theme API | **STILL OPEN** | Options struct or `Cell` + `CellStyle`; migrate ~92 call sites; optionally multi-line until API shrinks |
| 2 | `LayOut` / `LayOutFrom` options struct | **STILL OPEN** | Collapse 5–7 floats into `LayOutParams`; break remaining 4+ arg calls |
| 3 | Finish dual-assembler consolidation | **PARTIAL** | Share font/XMP emission; replace 8–10 param helpers with `fontChain`/`structureIDs`/`catalogOpts` structs |
| 4 | Finish `buildSubsetTTF` split | **PARTIAL** | Extract pack/dir/checksum; aim &lt;100 line orchestrator |
| 5 | Typed PDF values + encoding | **PARTIAL** | Standardize on `write.Ref` / `write.PDFString`; kill `%v` escape hatch |
| 6 | Unexport dead surface | **STILL OPEN** | `layout.Point`, `structure.Manager` (or wire it), theme colors if product-only |
| 7 | Hygiene pass | **PARTIAL** | Named multi-operand conditions, `range n`, non-nil slice init, multi-line XMP/ToUnicode, `%q` on paths |

---

## What is already strong

- Package map matches domain (font, layout, write, model, render).
- Top-level APIs use config/options structs.
- Leaf control flow (guards, switch, range) is idiomatic; assembler nesting improved.
- Stdlib-only, no reflection, careful value/pointer discipline.
- Sample programs stay thin wrappers around the engine.
- **New:** lint floor (`.golangci.yml`, `make lint`) and codehound presence keep complexity from regressing past funlen/gocyclo thresholds.

---

## Suggested score target after cleanup

| After step | Expected overall |
|------------|-----------------:|
| Today (2026-07-29) | **6.6** |
| Steps 1–2 (StyledCell + LayOut APIs) | **~7.4** |
| Steps 1–4 (API + assembler finish + subset) | **~8.0** |
| Steps 1–5 (encoding) | **~8.3–8.6** |
| Full hygiene | **~8.5–9.0** |

---

## Score delta section

| Axis | 2026-07-25 | 2026-07-29 | Δ |
|------|----------:|----------:|--:|
| Control flow | 6.5 | 7.0 | **+0.5** |
| Function design & variables | 5.5 | 6.0 | **+0.5** |
| Line length, breaking & file org | 5.5 | 5.8 | **+0.3** |
| Values, strings, types & philosophy | 7.4 | 7.5 | **+0.1** |
| **Overall** | **6.2** | **6.6** | **+0.4** |

### What drove the lift

- Assembler extraction: `Generate` 300→161, `GenerateDocument` 355→180, shared catalog helper.
- Subset decomposition: helpers for glyphs/loca/glyf/hmtx; early error returns for missing tables.
- Document font path early-return structure (`addFontChain`).
- Partial `write.Ref` usage and minor `interface{}` site reduction.
- Lint tooling makes further regression less likely.

### What held the score down

- `StyledCell` and `LayOut*` APIs unchanged (dominant line-breaking + param debt).
- Extraction created **new** 8–10 parameter helpers without options structs.
- PDF typing/encoding still mostly `map[string]interface{}` + sprintf refs.
- Dead exports and product theme colors still on the public surface.

---

## Agent attribution

| Pass | Dimension | Score |
|------|-----------|------:|
| Re-review | Control flow | 7.0 |
| Re-review | Function design & variable declarations | 6.0 |
| Re-review | Line length, breaking & code organization | 5.8 |
| Re-review | Values, strings, types & philosophy | 7.5 |

Skill source: `~/.agents/skills/golang-code-style` (samber/cc-skills-golang@golang-code-style).

Method note: This re-review is measurement-driven against the 2026-07-25 baseline (function lengths, long lines, param counts, prior-item greps) rather than four parallel first-look explore agents.

---

## Appendix — finding counts

| Dimension | High | Medium | Low | Total |
|-----------|-----:|-------:|----:|------:|
| Control flow | 0 | 10 | 4 | 14 |
| Function / variables | 2 | 11 | 3 | 16 |
| Line length / org | 2 | 5 | 2 | 9 |
| Values / strings / philosophy | 0 | 6 | 4 | 10 |
| **Combined (dedupe not applied)** | **4** | **32** | **13** | **49** |

Prior combined raw total was **60** (8 high / 35 medium / 17 low). High findings fell from **8 → 4** mainly by resolving dual-assembler nesting severity and shrinking god-function lengths below the “high” band.

Some findings appear in more than one dimension (e.g. `StyledCell` param count + call wrapping; dual assemblers length + helper params). Treat the priority list as the actionable merge, not the raw sum.

### Top remaining style debts (summary)

1. **`StyledCell` 7-arg API + ~92 one-line call sites**
2. **`LayOut*` / geometry helpers with 5–7 params**
3. **Assembler helper param explosion (8–10) without options structs**
4. **`buildSubsetTTF` still ~183-line orchestrator; subset.go 542 LOC**
5. **PDF AST as `map[string]interface{}` + mixed Ref/sprintf + unescaped strings**
6. **Dead exports (`layout.Point`, `structure.Manager`) and product theme colors in core**

---

*HTML companion: [golang-code-style-review-20260729.html](./golang-code-style-review-20260729.html)*
