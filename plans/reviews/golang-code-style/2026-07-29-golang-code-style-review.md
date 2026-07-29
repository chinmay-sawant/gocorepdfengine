# Go Code Style Review — gocorepdfengine (Follow-up)

| | |
|---|---|
| **Date** | 2026-07-29 |
| **Prior review** | 2026-07-25 — **6.2 / 10** |
| **Method** | Global skill `golang-code-style` (samber/cc-skills-golang@golang-code-style v1.2.2) |
| **Agents** | 4 dimension passes (control flow · function/vars · line length/org · values/strings/philosophy) |
| **Scope** | All `.go` under `engine/` and `sampledata/` |
| **Overall rating** | **6.7 / 10** (**+0.5**) |
| **Note on performance** | ~2× ops/sec is **product excellence**, not a style-axis win — see Architecture review |
| **Tool** | Grok Build |

---

## Performance context (not scored as style)

Compliant Zerodha x10: **b4 mean 1156 → current mean 2349 ops/sec** (~**2.0×**), best **2530**. That belongs to **architecture delivery/leverage** and a product-throughput side score (**8.5**), not to line-wrapping or param-count axes. Style still scores control flow / function shape / wrapping / types. PERF work did help style where it shortened assemblers and flattened hot paths (already in the +0.5).

---

## Executive summary

Since 2026-07-25, the biggest style win is **decomposing the dual PDF assemblers**. `GenerateDocument` is no longer a 350-line wall of IDs/streams/fonts/structure; it orchestrates five focused helpers with early exits. `Generate` gained `addFontObjects` and is flatter on the font path. Subset packing and the Zerodha bench are shorter for the same reason. Catalog paths now use `write.Ref` (zero remaining `fmt.Sprintf("%d 0 R")`).

What still anchors the score below ~7.5 is **API shape and presentation**, not missing packages: `StyledCell` (7 params, **74** one-line call sites), `LayOut`/`LayOutFrom` (5–7 params), and a wave of **new helpers that traded body length for parameter lists** (`addFontObjects` 10, `buildStructureTree` ~11, `buildCatalog` 9). Encoding philosophy remains strong (stdlib-only, no reflect) but the PDF object model is still `map[string]interface{}` (~76).

**Headline score: 6.7 / 10** — clearer control flow and organization; function-surface and wrapping debt still concentrated and mechanical.

---

## Scorecard (out of 10)

| Axis | Prev | **Now** | Δ | One-line justification |
|------|-----:|--------:|--:|------------------------|
| Control flow | 6.5 | **7.2** | +0.7 | Assembler happy paths flattened via extract+early return; leaf multi-operand / C-style loops remain |
| Function design & variables | 5.5 | **6.0** | +0.5 | Bodies shorter; debt shifted into 8–11-param helpers; `StyledCell`/`LayOut` still 5–7 |
| Line length, breaking & file org | 5.5 | **5.9** | +0.4 | Better file decomposition; systemic 7-arg one-liners + mega XMP literal unchanged |
| Values, strings, types & philosophy | 7.4 | **7.6** | +0.2 | `write.Ref` in catalog; zero deps; `interface{}` AST + unescaped PDF strings still dominant |
| **Weighted overall** | **6.2** | **6.7** | **+0.5** | Equal weight average |

### How the overall was computed

Equal weights (25% each):

```
7.2×0.25 + 6.0×0.25 + 5.9×0.25 + 7.6×0.25
= 1.80 + 1.50 + 1.475 + 1.90
= 6.675 → 6.7
```

### Rating band guide

| Band | Meaning |
|------|---------|
| 8–10 | Short focused functions, options structs, early returns, named composites, narrow types |
| 6–7 | Readable overall; known concentrated style debt |
| 4–5 | Widespread nesting, long param lists, unclear organization |
| ≤3 | Unreadable / hostile to maintenance |

**This codebase sits in the upper “readable with concentrated debt” band.**

---

## Delta vs 2026-07-25

| Prior issue | Status | Evidence |
|-------------|--------|----------|
| `GenerateDocument` ~355-line god body | **Mostly fixed** | Body now **~169 lines**; phases call helpers |
| Dual assembler nesting (high) | **Mostly fixed** | Helpers: `buildContentStreams`, `setupDocumentFont`, `buildStructureTree`, `addComplianceMetadata`, `buildCatalog` |
| `Generate` ~300-line font nest | **Improved** | Body **~207 lines**; font path extracted to `addFontObjects` |
| `buildSubsetTTF` ~280 lines | **Improved** | **~195 lines**; still a large packer |
| `runBenchmark` ~245 lines | **Improved** | **~191 lines**; helpers extracted |
| `fmt.Sprintf("%d 0 R")` (~30) | **Resolved** | **0** matches; catalog/engine use `write.Ref` |
| `write.Ref` underuse | **Partial** | Catalog path fixed; **~26** `" 0 R"` concatenations remain in page/structure/font/pdfa |
| `StyledCell` 7-arg one-liners (~92) | **Unchanged** | **74** call sites (still all one-line) |
| `map[string]interface{}` (~77) | **Unchanged** | **~76** sites |
| Zero deps / no `reflect` | **Still strong** | Unchanged |

**Net:** Assembler **length/nesting** and **ref formatting in the hot catalog path** improved. Public layout API shape, call wrapping, and the `interface{}` PDF AST did **not**.

---

## What was reviewed

| Area | Path | Style focus |
|------|------|-------------|
| Assemblers | `engine/engine.go`, `engine/document.go` | Length, nesting, duplication, helpers |
| Layout / theme | `engine/layout/*` | Param counts, call wrapping, loops |
| Font pipeline | `engine/font/*` | Subset, switch, range, strings |
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

### 1. Control flow — **7.2 / 10**

#### Current findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | medium | `layout/table.go` `layOutFrom` | Nesting | Image decode success still nested under `if err == nil` |
| 2 | medium | `render/template.go` | Nesting / else | First-table vs continue; color default `else if` chains |
| 3 | medium | `font/metrics.go` `BuildCIDToGIDMap` | Nesting | Nested map-lookup branches |
| 4 | medium | `color/hex.go` `ParseHex` | Named booleans | `err1 \|\| err2 \|\| err3` |
| 5 | medium | `image/image.go` `NewFromJPEG` | Named booleans | Multi-operand marker skip condition |
| 6 | medium | `model/contract_note.go` `ExpandTrades` | Named booleans | 3-operand retail keep guard |
| 7 | medium | `render/template.go` | Named booleans | JPEG magic multi-operand check |
| 8 | medium | `font/ttf.go` `parseName` | Prefer switch | `platformID` if / else-if |
| 9 | medium | `font/ttf.go` `StemVValue` | Drop else | `else if` after threshold |
| 10 | low | subset/props/model/bench | Prefer `range n` | C-style count loops (~8 sites) |
| 11 | low | content/write/structure | Minor else | Optional flatten |

#### Positive patterns

- Early guards widespread (`GenerateDocument` empty defaults, structure `Manager`, font subset length checks).
- Helpers use early return for disabled modes (`buildStructureTree` if `!isUA`).
- `switch` for alignment, JPEG components, escape/type encoding.
- `range` is the default over collections in layout/render/document.

#### Stats

| Severity | Count |
|----------|------:|
| high | 0 |
| medium | 9 |
| low | 2 |
| **total** | **11** |

---

### 2. Function design & variable declarations — **6.0 / 10**

#### Length metrics

| Function | Prior | Now | Lines (approx.) |
|----------|------:|----:|----------------:|
| `GenerateDocument` | ~355 | **~169** | `document.go` 63–231 |
| `Generate` | ~300 | **~207** | `engine.go` 97–303 |
| `buildSubsetTTF` | ~280 | **~195** | `subset.go` 430–624 |
| `runBenchmark` | ~245 | **~191** | `bench.go` 213–403 |
| `addFontObjects` | — | **~85** | `engine.go` 305–389 |
| `setupDocumentFont` | — | **~63** | `document.go` 292–354 |

#### Param counts (≤4 rule)

| Function | Params | Sev |
|----------|-------:|-----|
| `addFontObjects` | **10** | high |
| `buildStructureTree` | **~11** | high |
| `runWarmup` | **12** | high |
| `buildCatalog` | **9** | medium |
| `addComplianceMetadata` | **8** | medium |
| `collectGlyphInfos` | **7** | medium |
| `StyledCell` | **7** | medium |
| `layOutFrom` | **7** | medium |
| `LayOut` / `LayOutFrom` | **5–6** | medium |
| `PlaceImage` | **6** | medium |
| `cellFromProps` | **6** | medium |
| `spanProps` | **5** | medium |
| `buildSubsetTTF` | **5** | medium |

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | **high** | `engine.go` `addFontObjects` | ≤4 params / options | **10** params — extract fixed body length by exploding signature |
| 2 | **high** | `document.go` `buildStructureTree` | ≤4 params | Bundle IDs into `structureIDs` / ctx struct |
| 3 | **high** | `font/subset.go` `buildSubsetTTF` | Short/focused | Still ~195-line pack+checksum |
| 4 | medium | `layout/theme.go` `StyledCell` | ≤4 params | 7 params; 74 call sites |
| 5 | medium | `layout/table.go` `LayOut*` | ≤4 params | Geometry should be `LayOutParams` |
| 6 | medium | `sampledata/.../bench.go` `runWarmup` | ≤4 params | 12 params |
| 7 | medium | `structure/manager.go` `Build` | Result struct | 7 named return values |
| 8 | medium | `model/template.go` `PageSizes` | Named fields | Positional `{842, 1191}` |
| 9 | medium | Dual font setup | Short/focused | `addFontObjects` ≈ `setupDocumentFont` duplication |
| 10–13 | low/medium | matrix APIs, nil slices | polish | Incremental |

#### Positive patterns

- Public options: `engine.Config`, `DocumentConfig`, `render.Options`.
- Capacity-aware `make` for `contentIDs` / `pageIDs` / builders.
- Font parse still well-split (`parseHead`, cmap, OS/2, …).
- Contract note entry stays thin: model → layout → `GenerateDocument`.

#### Stats

| Severity | Count |
|----------|------:|
| high | 3 |
| medium | 8 |
| low | 2 |
| **total** | **13** |

---

### 3. Line length, breaking & code organization — **5.9 / 10**

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | **high** | `layout/theme.go` + **74** call sites | 4+ args one-per-line | Every `StyledCell(...)` is a single line |
| 2 | medium | `LayOut` / `PlaceImage` / `cellFromProps` / `spanProps` | 4+ args / options | Signatures and many call sites remain dense |
| 3 | medium | `meta/xmp.go` `xmpTemplateStr` | Break long lines | Entire XMP template is **one** enormous string literal |
| 4 | medium | `document.go` / `engine.go` identity ToUnicode | Break long lines | Multi-hundred-char fallback CMap on one line |
| 5 | medium | Dual generate paths | Group related | Font/catalog logic still parallel in two files |
| 6 | medium | Export surface | Unexport aggressively | `layout.Point` unused; `structure.Manager` unused outside package |
| 7 | medium | `structure.Manager.Build` | Result struct | 7-tuple return |
| 8 | low | Multi-concern files | One primary type | Acceptable for small packages |

**Blank imports:** 0 · **Dot imports:** 0 — compliant.

#### Counts

| Metric | Value |
|--------|------:|
| `StyledCell` call sites | **74** (was ~92) |
| `map[string]interface{}` | **~76** |
| Blank / dot imports | **0** |
| Lines >120 (sampledata) | **0** (effectively) |
| Lines >120 (engine) | concentrated in XMP + ToUnicode + StyledCell lines |

---

### 4. Values, strings, types & philosophy — **7.6 / 10**

#### Findings

| # | Sev | Location | Rule | Notes |
|---|-----|----------|------|-------|
| 1 | medium | Cross-cutting PDF dicts | Avoid `any` / narrow types | **~76×** `map[string]interface{}`; encoder `%v` fallback remains |
| 2 | medium | page/structure/font/pdfa | Prefer `write.Ref` | **~26** `" 0 R"` string builds; catalog path already uses `write.Ref` |
| 3 | medium | structure Title/Alt/Lang; catalog Lang | PDF string escape | `"(" + s + ")"` without `write.PDFString` |
| 4 | medium | `font/emit.go` `/W` / bbox | Explicit encode | `[]int` relies on Go `%v` list shape |
| 5 | medium | `document.go` footer hex | strconv in loops | `fmt.Fprintf(&buf, "%04X", r)` per rune |
| 6 | medium | `model` Load* errors | `%q` | Paths often still `%s` |
| 7 | medium | `color` Theme* vars | Minimize public surface | Product palette still exported from core `engine/color` |
| 8 | low | `CIDSystemInfo` Registry/Ordering | PDFString | unescaped parens |

#### Positive patterns

- **Zero third-party deps** (`go.mod`).
- **No `reflect`**.
- Value vs pointer discipline sound.
- `write.PDFString` used correctly in `pdfa.OutputIntentDict`.
- `write.Ref` on catalog/trailer paths.
- `hex04` + `strconv` for ToUnicode ranges; `strconv.Append*` in footers/page nums.

---

## Finding volume

| Dimension | High | Medium | Low | Total |
|-----------|-----:|-------:|----:|------:|
| Control flow | 0 | 9 | 2 | 11 |
| Function / variables | 3 | 8 | 2 | 13 |
| Line length / org | 1 | 6 | 1 | 8 |
| Values / strings / philosophy | 0 | 7 | 1 | 8 |
| **Combined (no cross-dedupe)** | **4** | **30** | **6** | **40** |

Prior combined was **60** (8 high). High findings **halved** (8 → 4); total volume down mainly from resolving assembler nesting and `fmt.Sprintf("%d 0 R")`.

---

## Priority cleanup order

1. **`StyledCell` / theme API** — options struct or `Cell{Text, Style}` builder; fix **74** call sites + wrapping.
2. **`LayOut` / `LayOutFrom` options** — `LayOutParams{MarginLeft, MarginTop, PageW, PageH, Y, Start}`; collapse 5–7 float APIs.
3. **Bundle assembler helper IDs** — `fontObjectIDs`, `structureIDs`, compliance refs; target ≤4 params on `addFontObjects` / `buildStructureTree` / `buildCatalog`.
4. **Share font emission** between `addFontObjects` and `setupDocumentFont` (one implementation).
5. **Finish `buildSubsetTTF` split** — table mutate vs directory write vs checksum.
6. **Typed PDF values + uniform refs** — `write.Ref` everywhere; `write.PDFString` for Lang/Title/Alt; shrink `%v` path.
7. **Unexport dead surface** — `layout.Point`, unused `structure.Manager` (or wire it), product Theme colors if appropriate.
8. **Hygiene** — named multi-operand conditions, `range n`, non-nil slice init, multi-line XMP/ToUnicode, `%q` on paths.

---

## What is already strong

- Clear domain package map (`content`, `font`, `layout`, `write`, `structure`, `pdfa`, …).
- Top-level generation APIs use config structs.
- Post-extraction control flow on assemblers is early-return-friendly.
- Stdlib-only, no reflection, careful value/pointer use.
- Catalog/trailer paths standardized on `write.Ref`.
- PDF/A helpers use `write.PDFString`.
- Samples stay thin drivers around `render` / `GenerateDocument`.
- Subset pipeline already has useful mid-level helpers.

---

## Suggested score target after cleanup

| After step | Expected overall |
|------------|-----------------:|
| Today | **6.7** |
| Steps 1–3 (API + param bundles) | **~7.6–7.8** |
| Steps 1–5 (shared font + subset split) | **~8.0–8.3** |
| Steps 1–6 (typed PDF / Ref / PDFString) | **~8.4–8.7** |
| Full hygiene (7–8) | **~8.7–9.0** |

---

## Agent attribution

| Agent | Dimension | Score |
|-------|-----------|------:|
| 1 | Control flow | **7.2** |
| 2 | Function design & variable declarations | **6.0** |
| 3 | Line length, breaking & file organization | **5.9** |
| 4 | Values, strings, types & philosophy | **7.6** |
| — | **Equal-weight overall** | **6.7** |

Skill source: `~/.agents/skills/golang-code-style` (samber/cc-skills-golang@golang-code-style).

---

## Appendix — finding counts

| Dimension | High | Medium | Low | Total |
|-----------|-----:|-------:|----:|------:|
| Control flow | 0 | 9 | 2 | 11 |
| Function / variables | 3 | 8 | 2 | 13 |
| Line length / org | 1 | 6 | 1 | 8 |
| Values / strings / philosophy | 0 | 7 | 1 | 8 |
| **Combined (dedupe not applied)** | **4** | **30** | **6** | **40** |

### Quick evidence

```
GenerateDocument body:     ~169 lines (helpers extracted)
Generate body:             ~207 lines (+ addFontObjects)
buildSubsetTTF:            ~195 lines
runBenchmark:              ~191 lines
StyledCell call sites:     74 (+ 1 def)
map[string]interface{}:    ~76
fmt.Sprintf("%d 0 R"):     0 (resolved)
write.Ref usages:          catalog/trailer paths
" 0 R" concat remaining:   ~26 (page, structure, font/emit, pdfa)
external deps:             0
reflect:                   0
```

## Related artifacts

- HTML visual report: [`golang-code-style-review-20260729-121831.html`](./golang-code-style-review-20260729-121831.html)
- Prior review: [`2026-07-25-golang-code-style-review.md`](./2026-07-25-golang-code-style-review.md)
