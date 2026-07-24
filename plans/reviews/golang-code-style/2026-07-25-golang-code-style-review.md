# Go Code Style Review — gocorepdfengine

| | |
|---|---|
| **Date** | 2026-07-25 |
| **Method** | Global skill `golang-code-style` (samber/cc-skills-golang@golang-code-style v1.2.2) |
| **Agents** | 4 parallel explore passes (control flow · function/vars · line length/org · values/strings/philosophy) |
| **Scope** | All `.go` under `engine/` and `sampledata/` (~36 files, 16 packages) |
| **Overall rating** | **6.2 / 10** |
| **Tool** | Grok Build |

---

## Executive summary

gocorepdfengine is a **stdlib-only** PDF engine with a clean package map and several focused leaf helpers. Style is strongest on **philosophy** (zero deps, no `reflect`, sound value-vs-pointer choices) and weakest where the skill cares about **function shape**: multi-hundred-line assemblers, 5–7 parameter APIs (`StyledCell`, `LayOut*`), and one-line call sites that violate the 4+ args rule.

Control flow is **generally good** at the leaf level (early returns, sensible `switch`/`range`), but the dual generate paths (`Generate` / `GenerateDocument`) bury happy-path font emission under nested mode/fallback branches. String handling is mixed: solid `strconv`/`strings.Builder` islands, but widespread `fmt.Sprintf("%d 0 R")` and some unescaped PDF string literals.

**Headline score: 6.2 / 10** — works and is readable in small units; style debt is concentrated and mechanical to fix (options structs, extract helpers, multi-line args, typed PDF values).

---

## Scorecard (out of 10)

| Axis | Score | One-line justification |
|------|------:|------------------------|
| Control flow | **6.5** | Strong early-return/`switch`/`range` habits; dual assemblers + multi-operand conditions drag score. |
| Function design & variables | **5.5** | Options structs at public entry points; god-functions and 5–7 param helpers dominate debt. |
| Line length, breaking & file org | **5.5** | Most lines ≤120; systemic 4+ arg one-liners (~92 `StyledCell` calls) and mega-literals. |
| Values, strings, types & philosophy | **7.4** | Zero deps / no reflect / good pointers; `interface{}` PDF AST and ref/hex `fmt` weak spots. |
| **Weighted overall** | **6.2** | Equal weight average of the four axes, rounded. |

### How the overall was computed

Equal weights (25% each):

```
6.5×0.25 + 5.5×0.25 + 5.5×0.25 + 7.4×0.25
= 1.625 + 1.375 + 1.375 + 1.85
= 6.225 → 6.2
```

### Rating band guide

| Band | Meaning |
|------|---------|
| 8–10 | Short focused functions, options structs, early returns, named composites, narrow types |
| 6–7 | Readable overall; known concentrated style debt |
| 4–5 | Widespread nesting, long param lists, unclear organization |
| ≤3 | Unreadable / hostile to maintenance |

**This codebase sits in the lower “readable with concentrated debt” band.**

---

## What was reviewed

| Area | Path | Style focus |
|------|------|-------------|
| Assemblers | `engine/engine.go`, `engine/document.go` | Length, nesting, duplication |
| Layout / theme | `engine/layout/*` | Param counts, call wrapping, loops |
| Font pipeline | `engine/font/*` | Subset monolith, switch, range, strings |
| Render / model | `engine/render/*`, `engine/model/*` | Cell APIs, positional literals |
| Encode / write | `engine/write/*`, `engine/doc/*` | `interface{}`, file order, refs |
| Structure / pdfa / meta / color / image / page / content | package files | Conditions, exports, strings |
| Samples | `sampledata/**` | Bench length, range loops |

Skill rules applied (not naming/lint/godoc — those are sibling skills):

- Line length & breaking (~120; 4+ args one-per-line)
- Variable declarations (`:=` vs `var`; non-nil slices/maps; named composite fields)
- Control flow (early return; no useless else; named multi-operand conditions; switch; range)
- Function design (short; ≤4 params; context order; naked returns)
- Value vs pointer args
- Code organization within files
- String handling & type conversions
- Philosophy (deps, reflect, public surface)

---

## Dimension reports

### 1. Control flow — **6.5 / 10**

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | **high** | `engine/engine.go` `Generate` | Reduce nesting | Font path: `isA4` → load → `loadedFont != nil` happy path deeply nested |
| 2 | **high** | `engine/document.go` `GenerateDocument` | Reduce nesting | Same nested font/mode pattern as `Generate` |
| 3 | medium | `font/subset.go` `buildSubsetTTF` | Early return | Success in `if`, error in `else` for `head`/`hhea` |
| 4 | medium | `font/metrics.go` `BuildCIDToGIDMap` | Nesting | Nested map-lookup branches |
| 5 | medium | `layout/table.go` `layOutFrom` | Nesting | Image place nested under success `err == nil` |
| 6 | medium | `render/template.go` | Nesting | Repeated `if err == nil { … }` color/image blocks |
| 7 | medium | `color/hex.go` `ParseHex` | Named booleans | `err1 \|\| err2 \|\| err3` |
| 8 | medium | `image/image.go` `NewFromJPEG` | Named booleans | Multi-operand SOI / marker checks |
| 9 | medium | `model/contract_note.go` `ExpandTrades` | Named booleans | 3-operand retail keep condition |
| 10 | medium | `render/template.go` | Named booleans | JPEG magic check |
| 11 | medium | `font/ttf.go` `parseName` | Prefer switch | `platformID` if/else-if |
| 12 | medium | `font/ttf.go` `StemVValue` | Drop else | `else if` after returnable branch |
| 13 | medium | subset/props/model/bench | Prefer `range n` | C-style count loops |
| 14–18 | low | content, write, structure, bench | Minor else / nesting | Optional cleanups |

#### Positive patterns

- Early guards common (`structure.Manager`, empty text, font load length checks, image markers).
- `switch` used well for escapes, object types, alignment, cmap formats, JPEG components.
- `range` over collections is the default in layout/render/document.
- Error-first I/O in sample mains is flat and clear.

#### Stats

| Severity | Count |
|----------|------:|
| high | 2 |
| medium | 11 |
| low | 5 |
| **total** | **18** |

---

### 2. Function design & variable declarations — **5.5 / 10**

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | **high** | `document.go` `GenerateDocument` (~355 lines) | Short/focused | IDs, streams, fonts, pages, meta, structure, catalog in one body |
| 2 | **high** | `engine.go` `Generate` (~300 lines) | Short/focused | Parallel single-page assembler |
| 3 | **high** | `font/subset.go` `buildSubsetTTF` (~280 lines) | Short/focused | Glyph extract + table rebuild + pack + checksum |
| 4 | **high** | `sampledata/zerodha/bench.go` `runBenchmark` (~245 lines) | Short/focused | Setup + workers + stats + I/O |
| 5 | medium | `layout/table.go` `layOutFrom` | ≤4 params | **7** params; public APIs 5–6 |
| 6 | medium | `layout/theme.go` `StyledCell` | ≤4 params | **7** params |
| 7 | medium | `render/template.go` `cellFromProps` | ≤4 params | **6** params |
| 8 | medium | `render/contract_note.go` `spanProps` | ≤4 params | **5** params |
| 9 | medium | `layout/layout.go` `PlaceImage` | ≤4 params | **6** params |
| 10 | medium | `font/subset.go` `buildSubsetTTF` | ≤4 params | **5** params |
| 11 | medium | `structure/manager.go` `Build` | Naked returns | Bare `return` with 7 named results in ~20-line body |
| 12 | medium | `model/template.go` `PageSizes` | Named fields | Positional `{842, 1191}` etc. |
| 13 | medium | `font/emit.go` `WidthsArray` | Named fields | `cidRange{cid, cid, width}` |
| 14 | medium | Multiple | Non-nil slices/maps | `var ranges []T`, nil `Kids`/`Elements` in constructors |
| 15 | medium | `render/contract_note.go` builders | Short/focused | ~80–90 line retail/active/HFT builders |
| 16–20 | low | misc | `var` misuse, speculative cap, matrix APIs | Minor |

`context.Context` is unused in scope — parameter-order rule is N/A.

#### Positive patterns

- Public options structs: `engine.Config`, `engine.DocumentConfig`, `render.Options`.
- Capacity-aware `make` when length is known (`contentIDs`, `pageIDs`).
- Named fields dominate exported composites.
- TTF parse split by table (`parseHead`, `parseCMap`, …) is a good counter-example to the subset/document monoliths.
- Contract-note entry stays thin: `PDF` → `BuildTable` → layout → `GenerateDocument`.

#### Stats

| Severity | Count |
|----------|------:|
| high | 4 |
| medium | 11 |
| low | 5 |
| **total** | **20** |

---

### 3. Line length, breaking & code organization — **5.5 / 10**

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | **high** | `layout/theme.go` + ~92 call sites | 4+ args one-per-line | `StyledCell` always one-line |
| 2 | **high** | `layout/table.go` `LayOut*` | >120 / options struct | 5–7 param signatures exceed ~120 |
| 3 | medium | PlaceImage, cellFromProps, spanProps, Tm/cm, buildSubsetTTF | 4+ args one-per-line | Geometry/matrix APIs |
| 4 | medium | `engine.go`, `document.go`, `meta/xmp.go` | Break long lines | Multi-hundred-char ToUnicode / XMP literals |
| 5 | medium | `write/writer.go` | Declaration order | `PDFString`/`Stream` interleaved with helpers |
| 6 | medium | Dual generate paths | Group related logic | Copy-pasted font/XMP/structure assembly |
| 7 | medium | Export surface | Unexport aggressively | Dead: `layout.Point`, `structure.Manager`; wide dict helpers |
| 8 | medium | `structure/manager.go` `Build` | Result struct | 7 return values |
| 9–10 | low | multi-type files; string notes | Org / strings | Optional |

**Blank imports:** 0 · **Dot imports:** 0 — compliant.

#### Positive patterns

- Package boundaries clear (`content`, `doc`, `font`, `layout`, `meta`, `model`, `page`, `pdfa`, `render`, `structure`, `write`).
- Font methods split by concern across files.
- Most production lines stay ≤120; long-line density is moderate and concentrated.
- `sampledata/` has **0** lines >120.

#### Stats

| Metric | Value |
|--------|------:|
| Lines >120 (`engine/`) | ~20 |
| Lines >120 (`sampledata/`) | 0 |
| `StyledCell` 7-arg one-line calls | ~92 |
| Blank / dot imports | 0 |
| Exported types in `engine/` | ~61 |
| Clearly unused exports | `layout.Point`, `structure.Manager` (+ related) |

---

### 4. Values, strings, types & philosophy — **7.4 / 10**

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | medium | Cross-cutting PDF dicts | Narrow types / avoid `any` | ~77× `map[string]interface{}`; encoder falls to `%v` |
| 2 | medium | `font/emit.go` widths / bbox | Explicit encode | `[]int` relies on Go `%v` looking like PDF |
| 3 | medium | page/structure/font/pdfa | strconv / helpers | ~30× `fmt.Sprintf("%d 0 R")` despite `write.Ref` |
| 4 | medium | TjCID, footer hex, ToUnicode | strconv in loops | `fmt.Sprintf("%04X")` in hot/loop paths |
| 5 | medium | StructElem Title/Alt; catalog Lang | PDF string escape | `fmt.Sprintf("(%s)", …)` without `PDFString` |
| 6 | medium | model Load*, some font errors | `%q` in errors | Paths often `%s` |
| 7 | medium | `color` Theme* vars | Minimize public surface | Product palette exported from core `engine/color` |
| 8–12 | low | StringLit buffer, appendStr, cacheKey, exports, DefaultBorder | polish | Incremental |

#### Positive patterns

- **Zero third-party deps** in `go.mod` — strong “little copying > little dependency.”
- **No `reflect`** in engine/sampledata.
- Small types by value; pointers for mutation, large/`nil`-meaningful cases.
- `strconv` for floats and display ints; `strings.Builder` in footer / `collectUsed`.
- `%q` already common in `color.ParseHex` and many font errors.

#### Stats

| Metric | Approx. |
|--------|---------|
| External deps | **0** |
| `reflect` usages | **0** |
| `map[string]interface{}` sites | **~77** |
| Pointless `*string`/`*int` params | **none** |

---

## Priority cleanup order

1. **Redesign `StyledCell` / theme API** — options struct or `Cell` + `CellStyle`; fix ~92 call sites.
2. **`LayOut` / `LayOutFrom` options struct** — collapse 5–7 floats into `LayOutParams`; break remaining 4+ arg calls.
3. **Split dual assemblers** — extract shared `emitFonts`, `emitStructure`, `identityToUnicode`, `buildCatalog`; flatten A-4 nesting with early branches.
4. **Split `buildSubsetTTF`** — collect glyphs / rebuild tables / assemble / checksum.
5. **Typed PDF values + encoding** — replace `%v` escape hatch; standardize on `write.Ref` / `write.PDFString`.
6. **Unexport dead surface** — `layout.Point`, `structure.Manager` (or wire it), theme colors if product-only.
7. **Hygiene pass** — named multi-operand conditions, `range n`, non-nil slice init, multi-line XMP/ToUnicode literals, `%q` on paths.

---

## What is already strong

- Package map matches domain (font, layout, write, model, render).
- Top-level APIs use config/options structs.
- Leaf control flow (guards, switch, range) is idiomatic.
- Stdlib-only, no reflection, careful value/pointer discipline.
- Sample programs stay thin wrappers around the engine.

---

## Suggested score target after cleanup

| After step | Expected overall |
|------------|-----------------:|
| Today | **6.2** |
| Steps 1–3 (API + assemblers) | **~7.5** |
| Steps 1–5 (encoding + subset split) | **~8.0–8.5** |
| Full hygiene | **~8.5–9.0** |

---

## Agent attribution

| Agent | Dimension | Score |
|-------|-----------|------:|
| 1 | Control flow | 6.5 |
| 2 | Function design & variable declarations | 5.5 |
| 3 | Line length, breaking & code organization | 5.5 |
| 4 | Values, strings, types & philosophy | 7.4 |

Skill source: `~/.agents/skills/golang-code-style` (samber/cc-skills-golang@golang-code-style).

---

## Appendix — finding counts

| Dimension | High | Medium | Low | Total |
|-----------|-----:|-------:|----:|------:|
| Control flow | 2 | 11 | 5 | 18 |
| Function / variables | 4 | 11 | 5 | 20 |
| Line length / org | 2 | 6 | 2 | 10 |
| Values / strings / philosophy | 0 | 7 | 5 | 12 |
| **Combined (dedupe not applied)** | **8** | **35** | **17** | **60** |

Some findings appear in more than one dimension (e.g. `StyledCell` param count + call wrapping; dual assemblers length + nesting). Treat the priority list as the actionable merge, not the raw sum.
