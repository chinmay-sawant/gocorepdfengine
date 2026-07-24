# gocorepdfengine — Ponytail Ultra Review

> **Generated:** 2026-07-25  
> **Mode:** Ultra (maximum aggression — find everything questionable)  
> **Method:** Ponytail skill (`@dietrichgebert/ponytail` / `.grok/skills/ponytail`) + 3 parallel explore subagents  
> **Scope:** `engine/**`, `sampledata/**`, compliance harness (high level)  
> **Overall rating:** **6.3 / 10** (ponytail leanness)

---

## Overall Rating

| Metric | Score | Notes |
|---|---|---|
| **Ponytail leanness (overall)** | **6.3 / 10** | Works, but first-order bloat + dual assembly + dead scaffolding |
| Dead API surface | 4.3 / 10 | Many unused exports, mode bits, content ops, Manager |
| Duplication | 5.0 / 10 | `Generate` ≈ `GenerateDocument`; dual render helpers; dual x10 scripts |
| Over-abstraction | 5.3 / 10 | Registry-for-one-font; dual compliance flags; unwired Manager/pdfa |
| Intentional shortcuts | 8.0 / 10 | Shared F1 font, `map[string]interface{}` dicts, dual product models |
| Dependency bloat | **10 / 10** | Zero third-party Go deps (`go.mod` is module + go version only) |
| Scaffolding debt | 3.5 / 10 | `structure.Manager`, unused `pdfa` helpers, tag zoo, orphan scripts |

### Rating by area

| # | Area | Rating | Items | ~Slop | Verdict |
|---|---|---:|---:|---:|---|
| 1 | Core assembly + encode | **6.4** | 22 | ~450L | Dual assemblers dominate; many dead write/content APIs |
| 2 | Feature modules (font/layout/image/render/model) | **6.7** | 28 | ~400L | Product-shaped; dead layout helpers + unused JSON knobs |
| 3 | Compliance + samples + harness | **5.7** | 18 | ~400L | Half live helpers, half phase-plan scaffolding |
| | **Weighted overall** | **6.3** | **68** | **~1,200L** | See scoring notes |

Weights: Area 1 25% · Area 2 50% · Area 3 25% (by relative code weight).

```
6.4×0.25 + 6.7×0.50 + 5.7×0.25
= 1.60 + 3.35 + 1.425
= 6.375 → 6.3
```

### Rating scale

| Score | Meaning |
|---|---|
| 9.0–10 | YAGNI-clean; only documented `ponytail:` ceilings remain |
| 8.0–8.9 | Lean for capability; localized third-order slop |
| 7.0–7.9 | Second-order slop: dead exports, duplicated helpers, thin wrappers |
| 6.0–6.9 | **← current (6.3)** First-order bloat: unused types, dual paths, scaffolding |
| < 6.0 | Over-engineered; speculative abstractions dominate |

### How to read tags

| Tag | Meaning |
|---|---|
| `delete:` | Dead code / unused flexibility. Replacement: nothing. |
| `stdlib:` | Hand-rolled thing the standard library already ships. |
| `native:` | Dependency or code doing what the platform already does. |
| `yagni:` | Abstraction with one implementation / config nobody sets / one caller. |
| `shrink:` | Same logic, fewer lines. |
| `ponytail:` | Intentional keep — document as ceiling, not debt. |

---

## Executive Summary

gocorepdfengine is a **stdlib-only** PDF 2.0 engine with a working product path (`model` → `render` → `GenerateDocument`) and real PDF/A-4 / minimal UA emit. Ponytail does not question that product value.

What it questions is **~1,200 lines of removable slop**:

1. **Dual assembly** — `Generate` (~300 LOC) nearly clones `GenerateDocument`; production only uses the latter via `render`.
2. **Dead surface** — unused content operators, write helpers, doc builder methods, mode bits that never gate, layout helpers with zero callers.
3. **Compliance scaffolding** — `structure.Manager` is 100% unwired; most of `pdfa` is unused; structure tag vocabulary is Document+P only.
4. **Schema theatre** — many model JSON fields accepted then ignored at emit.

**Bottom line:** **6.3/10** — capability is lean where it matters (font subset, layout, dual samples); assembly and compliance packages still carry phase-plan foreshadowing. Highest leverage: collapse `Generate`, delete `manager.go` + dead `pdfa` helpers, purge unused content/write APIs, drop ignored model fields.

**Dependency score is a perfect 10** — do not add libraries to “fix” this.

---

## What was reviewed

| Area | Paths | Agent |
|------|-------|-------|
| 1 Core assembly + encode | `engine/engine.go`, `document.go`, `doc/`, `write/`, `page/`, `content/` | Subagent A |
| 2 Feature modules | `engine/font/`, `layout/`, `image/`, `render/`, `model/` | Subagent B |
| 3 Compliance + harness | `engine/pdfa/`, `structure/`, `meta/`, `color/`, `sampledata/`, `compliance/` | Subagent C |

Total Go under review: **~6,622 LOC** (engine + sampledata). Zero third-party deps.

---

## 1. Core assembly + encode — Rating **6.4 / 10**

**Targets:** 7 files | **~450 lines removable** | **Items:** 22

### Checklist

- [ ] `yagni:` **Dual assembly path.** `Generate` (~300 LOC) rebuilds the entire PDF almost line-for-line with `GenerateDocument`. Production only calls `GenerateDocument` via `render`; `Generate` is only used from `engine_test.go` (zero callers under `sampledata/`). Collapse to a thin smoke wrapper that builds one `PageContent` and calls `GenerateDocument`, or retarget tests and delete `Generate`/`Config`/`Result`. [`engine/engine.go:26-341`] vs [`engine/document.go:43-398`]
- [ ] `delete:` **`engine.Result`.** One-field wrapper `{Data []byte}`; product API already returns `[]byte`. [`engine/engine.go:40-42`]
- [ ] `delete:` **Dead `fontNames` collection.** Built from `UsedFonts`/`FontRes`, then never read — emission always uses one shared `fontChain`. [`engine/document.go:77-89`]
- [ ] `delete:` **`PageContent.UsedFonts`.** Only consumed by the dead `fontNames` loop. `FontRes` alone drives resource labels. [`engine/document.go:24,80-82`]
- [ ] `delete:` **`fontChain.resourceLabel`.** Set to `"F1"` once, never read. [`engine/document.go:100-106`]
- [ ] `delete:` **`doc.ModeEmbedFonts`.** Set in both assemblers and in render but **never checked** (`HasMode(ModeEmbedFonts)` has zero call sites). [`engine/doc/builder.go:20`] [`engine/engine.go:53`] [`engine/document.go:68`]
- [ ] `delete:` **`Document.AddObject`.** Never called; only `AddObjectAt` is used. [`engine/doc/builder.go:56-60`]
- [ ] `delete:` **`Document.SetTrailerInfo`.** Zero callers; code mutates `TrailerInfo` directly. [`engine/doc/builder.go:77-79`]
- [ ] `delete:` **`SetPagesRoot` / `pagesRootRef`.** Written and never read in `Build()` (catalog gets `/Pages` from a local `pagesID`). [`engine/doc/builder.go:34,73-75`] [`engine/engine.go:250`] [`engine/document.go:319`]
- [ ] `delete:` **`Build` `[]byte` / `default` object branches.** No caller stores raw `[]byte` or non-dict/stream object data. [`engine/doc/builder.go:122-127`]
- [ ] `delete:` **`write.WriteComment`.** Zero callers. [`engine/write/writer.go:42-44`]
- [ ] `delete:` **`write.WriteTrailer`.** Dead; `Document.Build` hand-rolls trailer via `WriteDict`. [`engine/write/writer.go:116-127`]
- [ ] `delete:` **`write.Name`.** Defined, never referenced. [`engine/write/writer.go:141-143`]
- [ ] `delete:` **Half of `content.Stream` operators.** Layout mostly `fmt.Fprintf`s into `Stream.Buf` and only calls `BT`/`Tf`/`Td`/`TjCID`/`ET`/`Do` (plus `Generate` uses `BDC`/`EMC`/`Tj`). Never called: `Tm`, `TL`, `re`, `m`, `l`, `h`, `S`, `f`, `s`, `w`, `J`, `j`, `RG`, `rg`, `cm`, `q`, `Q`, `BMC`, `ArtifactBMC`. [`engine/content/content.go:74-169`]
- [ ] `delete:` **`content.Stream.Compress` + `Compressed`.** Zero callers; compress path for fonts is `compressData` in `engine.go`. [`engine/content/content.go:12,175-189`]
- [ ] `shrink:` **Footer/page-number emission reinventing operators.** Manual `strings.Builder` + hex CID loops instead of a short `content.Stream`. Duplicates near-identical footer / `Page N of M` blocks. [`engine/document.go:155-194`]
- [ ] `shrink:` **Duplicated PDF string escape.** `content.Tj` and `write.StringLit` implement the same escape rules. One helper. [`engine/content/content.go:39-64`] [`engine/write/writer.go:145-171`]
- [ ] `shrink:` **Duplicated A-4 font fallback + glyph alphabet.** Identical path in both assemblers; product path also skips `GenerateSubset` that smoke `Generate` runs. One shared `emitStandardFont(...)`. [`engine/engine.go:133-220`] [`engine/document.go:210-260`]
- [ ] `shrink:` **Duplicated catalog / XMP / ICC blocks.** Same `catalogDict` keys and dual `isA4`/`isUA` branches (~80 LOC each). Falls out of collapsing `Generate`. [`engine/engine.go:252-335`] [`engine/document.go:321-396`]
- [ ] `yagni:` **`Page.FontResources` / `XObjectResources` dual storage.** Assigned then ignored: `ToDict` takes separate map args. [`engine/page/page.go:12-13,34`] [`engine/document.go:280-312`]
- [ ] `yagni:` **`page` package is a pure dict serializer.** Keep only if the type is a deliberate ceiling; else `pagesDict(kids []ObjectID)`. [`engine/page/page.go:1-87`]
- [ ] `yagni:` **Empty-page default.** `len(cfg.Pages)==0` injects a blank stream; no production caller relies on this. [`engine/document.go:51-54`]
- [ ] `ponytail:` Keep **`map[string]interface{}` PDF dict encoding** — dynamic key sets fit a single `WriteDict`. [`engine/write/writer.go:46-96`]
- [ ] `ponytail:` Keep **pre-allocate IDs + `AddObjectAt`** — PDF graph needs forward refs. [`engine/doc/builder.go:50-67`]
- [ ] `ponytail:` Keep **single shared font (v1)** in `GenerateDocument` — product ceiling until multi-font embed is real. [`engine/document.go:97-116`]
- [ ] `ponytail:` Keep **`write.Encoder` as the byte sink**. [`engine/write/writer.go:13-40`]

#### Scorecard notes

| Dimension | Score |
|---|---:|
| Dead API surface | 4.0 |
| Duplication | 4.0 |
| Over-abstraction | 5.5 |
| Intentional shortcuts | 8.0 |

---

## 2. Feature modules — Rating **6.7 / 10**

**Targets:** 18 files | **~380–450 lines removable** | **Items:** 28

### Checklist

#### Layout

- [ ] `delete:` `WrapText` is fully dead — no callers in `engine/` or `sampledata/`. [`engine/layout/layout.go:66-108`]
- [ ] `delete:` `Point` type is unused. [`engine/layout/layout.go:11-13`]
- [ ] `delete:` `ContentBuilder.MCID` is written once, never read. [`engine/layout/layout.go:37,52`]
- [ ] `delete:` `ImageObj.Data` duplicates `Img.Data`; render only uses `obj.Img`. [`engine/layout/layout.go:41-44,169`]
- [ ] `delete:` No-op color assign after already setting it. [`engine/layout/table.go:160-162`]
- [ ] `delete:` `LayOut`/`LayOutFrom` always return `nil` error — drop `error` return. [`engine/layout/table.go:51-63,180`]
- [ ] `yagni:` `contentBottom` param always equals `marginTop`. Drop the 7th param. [`engine/layout/table.go:51-63`]
- [ ] `delete:` `CellProps.Bold`/`Italic`/`Underline` parsed then never applied. [`engine/layout/props.go:22-24,43-47`]
- [ ] `yagni:` `theme.go` is a one-caller stack, not a “theme system.” Fine as helpers. [`engine/layout/theme.go:1-30`]
- [ ] `ponytail:` Keep `textWidth = len*size*0.52` — deliberate lean shortcut vs full glyph metrics. Document ceiling. [`engine/layout/layout.go:62-64`]

#### Font

- [ ] `delete:` `Font.Widths`, `CharWidth`, `GlyphCount` — production-dead; only tests. [`engine/font/font.go:38-73`]
- [ ] `delete:` `IsTTF` — zero callers. [`engine/font/ttf.go:607-613`]
- [ ] `delete:` `AddChars` — production only uses `AddChar`. [`engine/font/subset.go:28-32`]
- [ ] `delete:` `UsedChars` is a pure alias of `UsedRunes`. [`engine/font/subset.go:34-36`]
- [ ] `delete:` `Family`, `IsSerif`, `IsMono` set during parse, never read. [`engine/font/font.go:10-13`]
- [ ] `delete:` `parseHMTX` / `parseMaxp` only existence-check; `parseGlyf` re-loads them. [`engine/font/ttf.go:128-131,200-211`]
- [ ] `delete:` `tableDir` + `entries` + `requiredTags` — map built, never consulted. [`engine/font/subset.go:52-65,90`]
- [ ] `delete:` `glyphIDArray` filled while every `idRangeOffset` forced to `0`. [`engine/font/subset.go:444-499`]
- [ ] `yagni:` `Registry` is not multi-font in practice — collapse to `LoadStandardFont(name string)`. [`engine/font/liberation.go:43-88`]
- [ ] `shrink:` `CIDSystemInfoDict` always called as `"Adobe","Identity",0` — inline the dict. [`engine/font/emit.go:28-29,59-65`]
- [ ] `ponytail:` Keep TTF load + subset bulk — required for PDF/A embed; no stdlib substitute. [`engine/font/ttf.go`] [`engine/font/subset.go`]

#### Image

- [ ] `delete:` `image.XObjectDict` unused in production; `document.go` inlines the same keys. [`engine/image/image.go:191-202`]
- [ ] `yagni:` Process-global SHA-256 image cache — no multi-doc daemon justifying permanent global map. [`engine/image/image.go:22-37,44-51`]
- [ ] `yagni:` PNG→JPEG re-encode for `w*h >= 100*100` is an undocumented quality tradeoff with no opt-out. [`engine/image/image.go:144-159`]
- [ ] `stdlib:` JPEG dimension parse reimplements SOF walking; `image.DecodeConfig` would shrink dimension extraction (keep raw bytes for `/DCTDecode`). [`engine/image/image.go:53-120`]

#### Render / model

- [ ] `shrink:` `scaleCols` (contract note) and `scaleColsWeights` (template) are the same loop. [`engine/render/contract_note.go:117-129`] [`engine/render/template.go:267-279`]
- [ ] `shrink:` Page-assembly loops (`ContentBuilder` → `engine.PageContent`) duplicated in `PDF` and `TemplatePDF`. [`engine/render/contract_note.go:49-61`] [`engine/render/template.go:53-65`]
- [ ] `delete:` `pageDimensions` is a one-line pass-through of `cfg.PageSize()`. [`engine/render/template.go:100-102`]
- [ ] `yagni:` `TemplatePDF(..., opts Options)` never reads `opts`. Drop param or honor `opts.Compliant`. [`engine/render/template.go:16-17,67-69`]
- [ ] `yagni:` `PDFACompliant` and `ArlingtonCompatible` OR into the same mode bits — dual flags for one behavior. [`engine/render/template.go:68-69`]
- [ ] `delete:` JSON schema fields never consumed by render: `Config.PageBorder`, `PageAlignment`, `EmbedFonts`; `Title.Link`; `TableCell.Link`/`Dest`; `Image.ImageName`; `Compliance`/`Features.Bookmarks`/`InternalLinks`; `Metadata.Keywords`; `Trade.Currency`; `Footer.Font`/`Link`; `Audit.AuditorSignaturePlaceholder`. [`engine/model/template.go`] [`engine/model/contract_note.go`]
- [ ] `yagni:` `BuildTable` exported but only used inside `render.PDF` — unexport. [`engine/render/contract_note.go:131-141`]
- [ ] `shrink:` `buildRetail` / `buildActive` / `buildHFT` (~260 lines) share header/section/alt-row patterns. [`engine/render/contract_note.go:152-411`]
- [ ] `ponytail:` Dual product surfaces (`ContractNote`/`PDF` vs `PDFTemplate`/`TemplatePDF`) mirror two sampledata products — intentional. Keep until products merge.

#### Scorecard notes

| Dimension | Score |
|---|---:|
| Dead API surface | 5.5 |
| Duplication | 6.0 |
| Over-abstraction | 6.5 |
| Intentional shortcuts | 8.5 |

---

## 3. Compliance + samples + harness — Rating **5.7 / 10**

**Targets:** ~16 primary files (+ compliance scripts) | **~380–450 lines removable** | **Items:** 18

### Package truth (live vs scaffolding)

| Package / area | Reality |
|---|---|
| `engine/meta` | **Live** — XMP wired from both assemblers |
| `engine/color` profiles + hex | **Live** for A-4 / layout theme (one dead theme var) |
| `engine/pdfa` | **Mostly dead** — only `OutputIntentDict` has callers |
| `engine/structure` dict helpers | **Live** for Document→P hand-assembly |
| `engine/structure.Manager` | **100% dead scaffolding** — zero external call sites |
| `sampledata/zerodha` | **Live harness**; x10 scripts duplicated |
| `compliance/` (root) | **Live** Makefile path |
| `compliance/verapdf/` | **Orphan** — superseded by root scripts |

### Checklist

- [ ] `delete:` Entire `structure.Manager` API is unused outside its own file. Assemblers hand-build Document→P instead. Delete `manager.go` (~82 LOC) or wire it. [`engine/structure/manager.go:1-81`]
- [ ] `delete:` `pdfa.CatalogA4Extras`, `PageResourceA4Extras`, `ImageColorSpace` have zero Go callers. Package is a 1-function helper pretending to be a policy module. [`engine/pdfa/pdfa.go:21-38`]
- [ ] `yagni:` 16 of 18 `StructType` constants never referenced outside `structure.go` (only `S_Document`/`S_P` used). Speculative tag zoo. [`engine/structure/structure.go:12-31`]
- [ ] `delete:` `NamespaceRef(id)` exported helper never called. [`engine/structure/structure.go:66-68`]
- [ ] `yagni:` `StructParentsValue` is identity `return pageIndex` — inline. [`engine/structure/structure.go:157-159`]
- [ ] `delete:` `OBJR` type + kid branch never constructed by any caller. [`engine/structure/structure.go:47-57,104-122`]
- [ ] `yagni:` `ParentTreeDict` second arg `annots` always `nil`; annot branch dead. [`engine/structure/structure.go:127-155`]
- [ ] `yagni:` `StructElem.Title` / `Alt` emission branches never populated by callers. [`engine/structure/structure.go:36-37,91-95`]
- [ ] `delete:` Dead const `xmpPacketPrefix` — string already hard-coded in template. [`engine/meta/xmp.go:39-41`]
- [ ] `yagni:` `XMPConfig.Keywords` never set by any caller. [`engine/meta/xmp.go:16,103`]
- [ ] `delete:` `color.ThemeLink` defined once, never used outside `hex.go`. [`engine/color/hex.go:62`]
- [ ] `shrink:` Hand-rolled ICC profile codec (~130 LOC) only as fallback when system ICC missing. Prefer `//go:embed` of two small blobs. [`engine/color/profiles.go:55-177`]
- [ ] `yagni:` Product Zerodha palette lives in core `engine/color` — only `contract_note` consumes it. [`engine/color/hex.go:49-65`]
- [ ] `delete:` Orphan `compliance/verapdf/` — Makefile uses root `compliance/run_verapdf.sh` + `verapdf_report.py` only. [`compliance/verapdf/`]
- [ ] `shrink:` `run_bench_x10.sh` and `run_bench_x10_nocomply.sh` near-clones. One script with `COMPLY=0|1`. [`sampledata/zerodha/run_bench_x10*.sh`]
- [ ] `stdlib:` `simpleRNG` xorshift reinvents `math/rand` for deterministic shuffle. [`sampledata/zerodha/bench.go:400-411`]
- [ ] `yagni:` Makefile alias targets `bench-zerodha-cached` / `uncached` / `bench-help` are one-liners over already-documented env. [`Makefile`]
- [ ] `yagni:` `structure_tree_check.py` validates ParentTree→TR/TD and empty `/Sect` — shapes this engine never emits. Forward-looking green gate. [`compliance/structure_tree_check.py`]
- [ ] `ponytail:` Build-tag split `main.go` / `main_nocomply.go` + shared `bench.go` is intentionally tiny — ceiling, not debt.
- [ ] `ponytail:` Live compliance path is real: `OutputIntentDict` + ICC + XMP + minimal structure dicts. Do not delete packages wholesale — delete unwired seams.

#### Scorecard notes

| Dimension | Score |
|---|---:|
| Dead API surface | 3.0 |
| Duplication | 5.0 |
| Over-abstraction | 4.0 |
| Intentional shortcuts | 7.0 |
| Scaffolding debt | 3.0 |

---

## Priority remediation (if cleaning)

Suggested order by **lines removed / risk**:

| Phase | Action | Est. Δ | Rating lift |
|---|---|---:|---|
| P0 | Collapse `Generate` → thin test helper over `GenerateDocument` | ~250L | +0.4 |
| P0 | Delete `structure/manager.go` + dead `pdfa` helpers + orphan `compliance/verapdf/` | ~120L | +0.3 |
| P1 | Purge unused content ops / write helpers / doc methods / dead fields | ~200L | +0.3 |
| P1 | Drop unused model JSON fields + always-nil layout errors + dead layout helpers | ~150L | +0.2 |
| P2 | Merge dual x10 bench scripts; unexport `BuildTable`; collapse Registry; inline identity helpers | ~100L | +0.1 |
| P2 | Shared `emitStandardFont` + footer via `content.Stream`; scaleCols helper | ~80L | +0.1 |
| | **Potential after full pass** | **~900–1,200L** | **→ ~7.5–7.8** |

Do **not** “fix” leanness by adding frameworks, DI containers, or interface-for-one-impl compliance profiles. Wire existing seams or delete them.

---

## Intentional ceilings (keep, label)

These are **not** debt if marked with a `// ponytail:` comment:

1. **`map[string]interface{}` PDF dicts** — dynamic Catalog/Page/Resources keys.
2. **Pre-allocated object IDs + `AddObjectAt`** — forward refs required by PDF graph.
3. **Single shared F1 font in `GenerateDocument`** — multi-font embed is a later product decision.
4. **Dual product models** (contract note vs financial template) — two real sample products.
5. **TTF parser + subsetter bulk** — PDF/A needs embed; no stdlib path.
6. **Heuristic text width** (`0.52 * size * len`) — full metrics only when alignment quality fails.
7. **Build-tag comply/nocomply split** — thin A/B harness for phase-8 benches.
8. **Document→P UA stub** — honest minimum until real table/sect tags land (then wire `Manager` or delete the façade permanently).

---

## Agents used

| Agent | Focus | Rating |
|---|---|---:|
| Subagent A (explore) | Core assembly + encode | 6.4 |
| Subagent B (explore) | Font / layout / image / render / model | 6.7 |
| Subagent C (explore) | Compliance + samples + harness | 5.7 |

Synthesis + weighted overall by orchestrator (this document).

---

## Bottom line

| | |
|---|---|
| **Overall ponytail rating** | **6.3 / 10** |
| Band | First-order bloat — works with known removable surface |
| Net removable (estimate) | **~1,200 lines** across 68 checklist items |
| Best single move | Collapse dual `Generate` / `GenerateDocument` assembly |
| Do not add | New deps, one-impl interfaces, more schema fields “for later” |

**Lean already on dependencies (10/10). Not lean yet on surface area (4.3/10 dead API).** Ship product features by wiring or deleting seams — never by scaffolding a second parallel path.
)
