# gocorepdfengine — Ponytail Ultra Review (Update)

> **Generated:** 2026-07-29  
> **Mode:** Ultra (maximum aggression — find everything questionable)  
> **Method:** Ponytail skill tag vocabulary (`delete:` / `stdlib:` / `native:` / `yagni:` / `shrink:` / `ponytail:`) + full re-walk of 2026-07-25 checklist against current tree  
> **Scope:** `engine/**`, `sampledata/**`, compliance harness (high level)  
> **Overall rating:** **6.5 / 10** (ponytail leanness)  
> **Prior (2026-07-25):** **6.3 / 10** · **Δ +0.2**

---

## Overall Rating

| Metric | Score | Notes |
|---|---|---|
| **Ponytail leanness (overall)** | **6.5 / 10** | Same first-order band; tiny lift from shared catalog helper + resourceLabel delete — dual assembly + dead Manager still dominate |
| Dead API surface | 4.5 / 10 | `resourceLabel` gone; watermark now calls `Tm`/`Q`; most prior dead exports remain; 11 content ops silenced with `//nolint:unused` |
| Duplication | 5.2 / 10 | `createCatalogDict` now shared; dual font-chain + dual structure-tree paths remain; dual x10 scripts remain |
| Over-abstraction | 5.4 / 10 | Registry-for-one-font; dual compliance flags; unwired Manager/pdfa |
| Intentional shortcuts | 8.0 / 10 | Shared F1 font, `map[string]interface{}` dicts, dual product models, full-font product embed |
| Dependency bloat | **10 / 10** | Still zero third-party Go deps (`go.mod` = module + `go 1.26.4` only) |
| Scaffolding debt | 3.6 / 10 | `structure.Manager`, unused `pdfa` helpers, tag zoo, orphan `compliance/verapdf/`, `_ = collectFontNames` dead call |

### Score delta (2026-07-25 → 2026-07-29)

| Metric | Old | New | Δ |
|---|---:|---:|---:|
| **Overall** | **6.3** | **6.5** | **+0.2** |
| Dead API surface | 4.3 | 4.5 | +0.2 |
| Duplication | 5.0 | 5.2 | +0.2 |
| Over-abstraction | 5.3 | 5.4 | +0.1 |
| Intentional shortcuts | 8.0 | 8.0 | 0 |
| Dependency bloat | 10 | 10 | 0 |
| Scaffolding debt | 3.5 | 3.6 | +0.1 |

### Rating by area

| # | Area | Rating | Items | ~Slop | Verdict |
|---|---|---:|---:|---:|---|
| 1 | Core assembly + encode | **6.6** | 22 | ~430L | Helper extract + shared catalog; dual assemblers + dead write/doc APIs remain |
| 2 | Feature modules (font/layout/image/render/model) | **6.8** | 28 | ~380L | Watermark uses Stream ops; dead layout/model surface largely untouched |
| 3 | Compliance + samples + harness | **5.7** | 18 | ~390L | Manager / pdfa scaffolding / orphan verapdf / dual x10 scripts unchanged |
| | **Weighted overall** | **6.5** | **68** | **~1,150L** | See scoring notes |

Weights: Area 1 25% · Area 2 50% · Area 3 25% (by relative code weight).

```
6.6×0.25 + 6.8×0.50 + 5.7×0.25
= 1.65 + 3.40 + 1.425
= 6.475 → 6.5
```

Area delta vs prior: Core 6.4→6.6 · Features 6.7→6.8 · Compliance 5.7→5.7.

### Rating scale

| Score | Meaning |
|---|---|
| 9.0–10 | YAGNI-clean; only documented `ponytail:` ceilings remain |
| 8.0–8.9 | Lean for capability; localized third-order slop |
| 7.0–7.9 | Second-order slop: dead exports, duplicated helpers, thin wrappers |
| 6.0–6.9 | **← current (6.5)** First-order bloat: unused types, dual paths, scaffolding |
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

Checklist status keys (this update):

| Mark | Meaning |
|---|---|
| **STILL OPEN** | Same debt as 2026-07-25; unchecked |
| **PARTIAL** | Some extraction / usage change; debt not closed |
| **FIXED** | Item no longer applies |

---

## Executive Summary

gocorepdfengine is still a **stdlib-only** PDF 2.0 engine with a working product path (`model` → `render` → `GenerateDocument`) and real PDF/A-4 / minimal UA emit. Since 2026-07-25 the tree gained a **golangci-lint gate**, mechanical refactors for complexity/funlen, a compliance-minded watermark fix, and small bench deltas — **not** a leanness pass.

What Ponytail still questions is **~1,150 lines of removable slop**:

1. **Dual assembly** — `Generate` (~160 LOC body + `buildA4FontChain` + single-page structure tree) still nearly clones product `GenerateDocument`. Production only uses `GenerateDocument` via `render`; `Generate` is tests-only. Shared win: `createCatalogDict` only.
2. **Dead surface** — unused content operators (now `//nolint:unused`), write helpers, doc methods, mode bits that never gate, layout helpers with zero callers. One real delete: `fontChain.resourceLabel`.
3. **Compliance scaffolding** — `structure.Manager` still 100% unwired; most of `pdfa` still unused; structure tag vocabulary is Document+P only.
4. **Schema theatre** — many model JSON fields accepted then ignored at emit.
5. **Lint papering** — `_ = collectFontNames(...)` and `//nolint:unused` on dead ops silence the gate without removing surface.

**Bottom line:** **6.5/10** (+0.2). Still first-order bloat. Highest leverage unchanged: collapse `Generate`, delete `manager.go` + dead `pdfa` helpers, purge unused content/write APIs, drop ignored model fields. Do **not** treat lint clean + bench noise as leanness.

**Dependency score remains a perfect 10** — do not add libraries to “fix” this.

---

## What was reviewed

| Area | Paths | Method |
|------|-------|--------|
| 1 Core assembly + encode | `engine/engine.go`, `document.go`, `doc/`, `write/`, `page/`, `content/` | Full re-grep + re-read of prior checklist |
| 2 Feature modules | `engine/font/`, `layout/`, `image/`, `render/`, `model/` | Call-site audit of every prior item |
| 3 Compliance + samples + harness | `engine/pdfa/`, `structure/`, `meta/`, `color/`, `sampledata/`, `compliance/` | Live vs scaffolding table + script orphans |

Total Go under review: **~6,691 LOC** (engine **6,201** + sampledata **490**). Prior was ~6,622 — **net +~69 LOC** despite “cleanups.” Zero third-party deps.

Notable process additions (not scored as leanness of product surface):

- `.golangci.yml` + `make lint` / `lint-all`
- `codehound.toml`
- Baseline/bench updates under `baselines/`

---

## 1. Core assembly + encode — Rating **6.6 / 10** (was 6.4)

**Targets:** 7 files | **~430 lines removable** | **Items:** 22

### Checklist

- [ ] **STILL OPEN** `yagni:` **Dual assembly path.** `Generate` still rebuilds a full PDF graph (IDs, font chain, page, A-4 meta, structure, catalog) parallel to `GenerateDocument`. Production only calls `GenerateDocument` via `render`; `Generate` only from `engine_test.go`. Partial improvement: both now call shared `createCatalogDict`. Collapse to a thin smoke wrapper over `GenerateDocument`, or retarget tests and delete `Generate`/`Config`/`Result`. [`engine/engine.go:124-282`] vs [`engine/document.go:48-226`]
- [ ] **STILL OPEN** `delete:` **`engine.Result`.** One-field wrapper `{Data []byte}`; product API already returns `[]byte`. [`engine/engine.go:120-122`]
- [ ] **STILL OPEN** `delete:` **Dead `collectFontNames`.** Prior dead `fontNames` loop is now extracted and **explicitly discarded**: `_ = collectFontNames(cfg.Pages)` — still never read; emission still uses one shared font chain. [`engine/document.go:81`] [`engine/document.go:340-354`]
- [ ] **STILL OPEN** `delete:` **`PageContent.UsedFonts`.** Only consumed by dead `collectFontNames`. `FontRes` alone drives resource labels. [`engine/document.go:24,343-345`]
- [x] **FIXED** `delete:` **`fontChain.resourceLabel`.** Removed in lint refactor (was set to `"F1"`, never read). No longer present on `fontChain`. [`engine/document.go:43-45`]
- [ ] **STILL OPEN** `delete:` **`doc.ModeEmbedFonts`.** Set in both assemblers and in render but **never checked** (`HasMode(ModeEmbedFonts)` has zero call sites). [`engine/doc/builder.go:20`] [`engine/engine.go:133`] [`engine/document.go:72`] [`engine/render/*`]
- [ ] **STILL OPEN** `delete:` **`Document.AddObject`.** Never called; only `AddObjectAt` is used. [`engine/doc/builder.go:56-60`]
- [ ] **STILL OPEN** `delete:` **`Document.SetTrailerInfo`.** Zero callers; code mutates `TrailerInfo` directly. [`engine/doc/builder.go:77-79`]
- [ ] **STILL OPEN** `delete:` **`SetPagesRoot` / `pagesRootRef`.** Written and never read in `Build()` (catalog gets `/Pages` from local `pagesID`). [`engine/doc/builder.go:34,73-75`] [`engine/engine.go:244`] [`engine/document.go:193`]
- [ ] **STILL OPEN** `delete:` **`Build` `[]byte` / `default` object branches.** No caller stores raw `[]byte` or non-dict/stream object data. [`engine/doc/builder.go:122-127`]
- [ ] **STILL OPEN** `delete:` **`write.WriteComment`.** Zero callers. [`engine/write/writer.go:43`]
- [ ] **STILL OPEN** `delete:` **`write.WriteTrailer`.** Dead; `Document.Build` hand-rolls trailer via `WriteDict`. [`engine/write/writer.go:117`]
- [ ] **STILL OPEN** `delete:` **`write.Name`.** Defined, never referenced. [`engine/write/writer.go:142`]
- [ ] **PARTIAL** `delete:` **Half of `content.Stream` operators.** Layout still mostly `fmt.Fprintf`s into `Stream.Buf` for fill/stroke/path; methods with **zero call sites** remain and are now papered with `//nolint:unused`: `re`, `m`, `l`, `h`, `f`, `s`, `w`, `j`, `rg`, `cm`, `q` (11). Live methods: `BT`/`ET`/`Tf`/`Td`/`Tj`/`TjCID`/`Tm`/`Do`/`BDC`/`BMC`/`EMC`/`Q`/`S`/`J`/`RG`/`ArtifactBMC` (partial — some only smoke/demo). Watermark now uses `Tm`/`Q`/`TjCID` (was raw fprintf) — small win. [`engine/content/content.go:84-180`] [`engine/layout/layout.go:129-190`]
- [ ] **STILL OPEN** `delete:` **`content.Stream.Compress` + `Compressed`.** Zero callers; compress path for fonts is `compressData` in `engine.go`. [`engine/content/content.go:12,186-200`]
- [ ] **STILL OPEN** `shrink:` **Footer/page-number emission reinventing operators.** Manual `strings.Builder` + hex CID loops instead of a short `content.Stream`. [`engine/document.go:234-273`]
- [ ] **STILL OPEN** `shrink:` **Duplicated PDF string escape.** `content.Tj` and `write.StringLit` implement the same escape rules. One helper. [`engine/content/content.go:39-64`] [`engine/write/writer.go:146+`]
- [ ] **STILL OPEN** `shrink:` **Duplicated A-4 font fallback + glyph alphabet.** `buildA4FontChain` (demo, **subsets**) vs `addFontChain`/`addLoadedFontChain` (product, **full RawData embed** — still no `GenerateSubset`). One shared `emitStandardFont(...)`. [`engine/engine.go:26-104`] [`engine/document.go:285-397`]
- [ ] **PARTIAL** `shrink:` **Duplicated catalog / XMP / ICC blocks.** Catalog dict emission now shared via `createCatalogDict`. XMP/ICC blocks and dual structure trees (`addSinglePageStructureTree` vs `addStructureTree`) still duplicated. [`engine/engine.go:246-316,318-339`] [`engine/document.go:195-221,309-337`]
- [ ] **STILL OPEN** `yagni:` **`Page.FontResources` / `XObjectResources` dual storage.** Assigned then ignored: `ToDict` takes separate map args (though `Generate` now passes `pg.FontResources` through). [`engine/page/page.go:12-13,34`] [`engine/document.go:154-186`]
- [ ] **STILL OPEN** `yagni:` **`page` package is a pure dict serializer.** Keep only if the type is a deliberate ceiling; else `pagesDict(kids []ObjectID)`. [`engine/page/page.go:1-87`]
- [ ] **STILL OPEN** `yagni:` **Empty-page default.** `len(cfg.Pages)==0` injects a blank stream; no production caller relies on this. [`engine/document.go:55-58`]
- [ ] `ponytail:` Keep **`map[string]interface{}` PDF dict encoding** — dynamic key sets fit a single `WriteDict`. [`engine/write/writer.go`]
- [ ] `ponytail:` Keep **pre-allocate IDs + `AddObjectAt`** — PDF graph needs forward refs. [`engine/doc/builder.go`]
- [ ] `ponytail:` Keep **single shared font (v1)** in `GenerateDocument` — product ceiling until multi-font embed is real. [`engine/document.go:89-103`]
- [ ] `ponytail:` Keep **`write.Encoder` as the byte sink**. [`engine/write/writer.go`]

#### Scorecard notes

| Dimension | Score | vs prior |
|---|---:|---:|
| Dead API surface | 4.2 | +0.2 |
| Duplication | 4.3 | +0.3 |
| Over-abstraction | 5.5 | 0 |
| Intentional shortcuts | 8.0 | 0 |

---

## 2. Feature modules — Rating **6.8 / 10** (was 6.7)

**Targets:** 18 files | **~380 lines removable** | **Items:** 28

### Checklist

#### Layout

- [ ] **STILL OPEN** `delete:` `WrapText` is fully dead — no callers in `engine/` or `sampledata/`. [`engine/layout/layout.go:66-108`]
- [ ] **STILL OPEN** `delete:` `Point` type is unused. [`engine/layout/layout.go:11-13`]
- [ ] **STILL OPEN** `delete:` `ContentBuilder.MCID` is written once, never read. [`engine/layout/layout.go:37,52`]
- [ ] **STILL OPEN** `delete:` `ImageObj.Data` duplicates `Img.Data`; render only uses `obj.Img`. [`engine/layout/layout.go:41-44,175`]
- [ ] **STILL OPEN** `delete:` No-op color assign after already setting it. [`engine/layout/table.go` ~160]
- [ ] **STILL OPEN** `delete:` `LayOut`/`LayOutFrom` always return `nil` error — drop `error` return. [`engine/layout/table.go:53-65`]
- [ ] **STILL OPEN** `yagni:` `contentBottom` param always equals `marginTop`. Drop the 7th param. [`engine/layout/table.go:54-76`]
- [ ] **STILL OPEN** `delete:` `CellProps.Bold`/`Italic`/`Underline` parsed then never applied. [`engine/layout/props.go:22-24,43-47`]
- [ ] **STILL OPEN** `yagni:` `theme.go` is a one-caller stack, not a “theme system.” Fine as helpers. [`engine/layout/theme.go:1-30`]
- [ ] `ponytail:` Keep `textWidth = len*size*0.52` — deliberate lean shortcut vs full glyph metrics. Document ceiling. [`engine/layout/layout.go:62-64`]

#### Font

- [ ] **STILL OPEN** `delete:` `Font.Widths`, `CharWidth`, `GlyphCount` — production-dead; only tests. [`engine/font/font.go:38-73`] [`engine/font/font_test.go`]
- [ ] **STILL OPEN** `delete:` `IsTTF` — zero callers. [`engine/font/ttf.go:604`]
- [ ] **STILL OPEN** `delete:` `AddChars` — production only uses `AddChar` (tests use `AddChars`). [`engine/font/subset.go:29`]
- [ ] **PARTIAL** `delete:` `UsedChars` — still a thin alias of `UsedRunes`, but **internally used** by subset path; production never calls the export. Can unexport. [`engine/font/subset.go:35,170`]
- [ ] **STILL OPEN** `delete:` `Family`, `IsSerif`, `IsMono` set during parse, never read. [`engine/font/font.go`] [`engine/font/ttf.go:179-180,411`]
- [ ] **STILL OPEN** `delete:` `parseHMTX` / `parseMaxp` only existence-check; `parseGlyf` re-loads them. [`engine/font/ttf.go:127-131,199-211`]
- [ ] **STILL OPEN** `delete:` `requiredTags` map is passed into `buildSubsetTTF` as `_ map[string]bool` — built, never consulted. [`engine/font/subset.go:58-67,209`]
- [ ] **STILL OPEN** `delete:` `glyphIDArray` still filled while range offsets forced to format that embeds the array. Keep if format-required; otherwise simplify. [`engine/font/subset.go:464+`]
- [ ] **STILL OPEN** `yagni:` `Registry` is not multi-font in practice — collapse to `LoadStandardFont(name string)`. [`engine/font/liberation.go:43-88`]
- [ ] **STILL OPEN** `shrink:` `CIDSystemInfoDict` always called as `"Adobe","Identity",0` — inline the dict. [`engine/font/emit.go:28,59`]
- [ ] `ponytail:` Keep TTF load + subset bulk — required for PDF/A embed; no stdlib substitute. **Note:** product `GenerateDocument` still embeds **full** `RawData` (subset only on demo `Generate`) — subset code is live capability but **unwired on the product path**. [`engine/font/ttf.go`] [`engine/font/subset.go`] [`engine/document.go:364`]

#### Image

- [ ] **STILL OPEN** `delete:` `image.XObjectDict` unused in production; `document.go` inlines the same keys. Only test calls it. [`engine/image/image.go:194`]
- [ ] **STILL OPEN** `yagni:` Process-global SHA-256 image cache — no multi-doc daemon justifying permanent global map. [`engine/image/image.go:22-55`]
- [ ] **STILL OPEN** `yagni:` PNG→JPEG re-encode for large images is an undocumented quality tradeoff with no opt-out. [`engine/image/image.go`]
- [ ] **STILL OPEN** `stdlib:` JPEG dimension parse reimplements SOF walking; `image.DecodeConfig` would shrink dimension extraction (keep raw bytes for `/DCTDecode`). [`engine/image/image.go`]

#### Render / model

- [ ] **STILL OPEN** `shrink:` `scaleCols` (contract note) and `scaleColsWeights` (template) are the same loop. [`engine/render/contract_note.go:123`] [`engine/render/template.go:271`]
- [ ] **STILL OPEN** `shrink:` Page-assembly loops (`ContentBuilder` → `engine.PageContent`) duplicated in `PDF` and `TemplatePDF`. [`engine/render/contract_note.go`] [`engine/render/template.go:53-65`]
- [ ] **STILL OPEN** `delete:` `pageDimensions` is a one-line pass-through of `cfg.PageSize()`. [`engine/render/template.go:104-106`]
- [ ] **STILL OPEN** `yagni:` `TemplatePDF(..., opts Options)` still ignores `opts` (`_ Options`). Drop param or honor `opts.Compliant`. [`engine/render/template.go:16`]
- [ ] **STILL OPEN** `yagni:` `PDFACompliant` and `ArlingtonCompatible` OR into the same mode bits — dual flags for one behavior. [`engine/render/template.go:68-69`]
- [ ] **STILL OPEN** `delete:` JSON schema fields never consumed by render: `Config.PageBorder`, `PageAlignment`, `EmbedFonts`; `Title.Link`; `TableCell.Link`/`Dest`; `Image.ImageName`; `Compliance`/`Features.Bookmarks`/`InternalLinks`; `Metadata.Keywords`; `Trade.Currency`; `Footer.Font`/`Link`; `Audit.AuditorSignaturePlaceholder`. [`engine/model/template.go`] [`engine/model/contract_note.go`]
- [ ] **STILL OPEN** `yagni:` `BuildTable` exported but only used inside `render.PDF` — unexport. [`engine/render/contract_note.go:137-138`]
- [ ] **STILL OPEN** `shrink:` `buildRetail` / `buildActive` / `buildHFT` share header/section/alt-row patterns. [`engine/render/contract_note.go`]
- [ ] `ponytail:` Dual product surfaces (`ContractNote`/`PDF` vs `PDFTemplate`/`TemplatePDF`) mirror two sampledata products — intentional. Keep until products merge.

#### Scorecard notes

| Dimension | Score | vs prior |
|---|---:|---:|
| Dead API surface | 5.6 | +0.1 |
| Duplication | 6.1 | +0.1 |
| Over-abstraction | 6.5 | 0 |
| Intentional shortcuts | 8.5 | 0 |

---

## 3. Compliance + samples + harness — Rating **5.7 / 10** (was 5.7)

**Targets:** ~16 primary files (+ compliance scripts) | **~390 lines removable** | **Items:** 18

### Package truth (live vs scaffolding)

| Package / area | Reality |
|---|---|
| `engine/meta` | **Live** — XMP wired from both assemblers |
| `engine/color` profiles + hex | **Live** for A-4 / layout theme (`ThemeLink` still dead) |
| `engine/pdfa` | **Mostly dead** — only `OutputIntentDict` has callers |
| `engine/structure` dict helpers | **Live** for Document→P hand-assembly |
| `engine/structure.Manager` | **100% dead scaffolding** — zero external call sites |
| `sampledata/zerodha` | **Live harness**; x10 scripts still duplicated |
| `compliance/` (root) | **Live** Makefile path |
| `compliance/verapdf/` | **Orphan** — superseded by root scripts |
| `tools/structure_tree_check.py` | **Near-dup** of `compliance/structure_tree_check.py` (files differ) |

### Checklist

- [ ] **STILL OPEN** `delete:` Entire `structure.Manager` API is unused outside its own file. Assemblers hand-build Document→P instead. Delete `manager.go` (~74 LOC) or wire it. [`engine/structure/manager.go:1-74`]
- [ ] **STILL OPEN** `delete:` `pdfa.CatalogA4Extras`, `PageResourceA4Extras`, `ImageColorSpace` have zero Go callers. Package is a 1-function helper pretending to be a policy module. [`engine/pdfa/pdfa.go:21-38`]
- [ ] **STILL OPEN** `yagni:` 16 of 18 `StructType` constants never referenced outside `structure.go` (only `SDocument`/`SP` used in assemblers). Speculative tag zoo. [`engine/structure/structure.go:12-31`]
- [ ] **STILL OPEN** `delete:` `NamespaceRef(id)` exported helper never called. [`engine/structure/structure.go:66`]
- [ ] **STILL OPEN** `yagni:` `StructParentsValue` is identity `return pageIndex` — inline. Still called from both assemblers. [`engine/structure/structure.go:158`] [`engine/document.go:183`] [`engine/engine.go:234`]
- [ ] **STILL OPEN** `delete:` `OBJR` type + kid branch never constructed by any caller. [`engine/structure/structure.go`]
- [ ] **STILL OPEN** `yagni:` `ParentTreeDict` second arg `annots` always `nil` from assemblers; annot branch dead. [`engine/structure/structure.go:128`] [`engine/document.go:335`]
- [ ] **STILL OPEN** `yagni:` `StructElem.Title` / `Alt` emission branches never populated by callers. [`engine/structure/structure.go`]
- [x] **FIXED** `delete:` Dead const `xmpPacketPrefix` — no longer present (template string embeds prefix). Prior item closed by earlier cleanup / not reintroduced.
- [ ] **STILL OPEN** `yagni:` `XMPConfig.Keywords` never set by any caller. [`engine/meta/xmp.go:16,101`]
- [ ] **STILL OPEN** `delete:` `color.ThemeLink` defined once, never used outside `hex.go`. [`engine/color/hex.go:62`]
- [ ] **STILL OPEN** `shrink:` Hand-rolled ICC profile codec (~130 LOC of profiles.go) only as fallback when system ICC missing. Prefer `//go:embed` of two small blobs. [`engine/color/profiles.go`]
- [ ] **STILL OPEN** `yagni:` Product Zerodha palette lives in core `engine/color` — only `contract_note` consumes it. [`engine/color/hex.go`]
- [ ] **STILL OPEN** `delete:` Orphan `compliance/verapdf/` — Makefile uses root `compliance/run_verapdf.sh` + `verapdf_report.py` only. [`compliance/verapdf/`]
- [ ] **STILL OPEN** `shrink:` `run_bench_x10.sh` and `run_bench_x10_nocomply.sh` near-clones. One script with `COMPLY=0|1`. [`sampledata/zerodha/run_bench_x10*.sh`]
- [ ] **STILL OPEN** `stdlib:` `simpleRNG` xorshift reinvents `math/rand` for deterministic shuffle. [`sampledata/zerodha/bench.go:407-411`]
- [ ] **STILL OPEN** `yagni:` Makefile alias targets `bench-zerodha-cached` / `uncached` / `bench-help` are one-liners over already-documented env. [`Makefile`]
- [ ] **STILL OPEN** `yagni:` `structure_tree_check.py` validates ParentTree→TR/TD and empty `/Sect` — shapes this engine never emits. Forward-looking green gate. [`compliance/structure_tree_check.py`]
- [ ] `ponytail:` Build-tag split `main.go` / `main_nocomply.go` + shared `bench.go` is intentionally tiny — ceiling, not debt.
- [ ] `ponytail:` Live compliance path is real: `OutputIntentDict` + ICC + XMP + minimal structure dicts. Do not delete packages wholesale — delete unwired seams.

#### Scorecard notes

| Dimension | Score | vs prior |
|---|---:|---:|
| Dead API surface | 3.0 | 0 |
| Duplication | 5.0 | 0 |
| Over-abstraction | 4.0 | 0 |
| Intentional shortcuts | 7.0 | 0 |
| Scaffolding debt | 3.0 | 0 |

---

## Priority remediation (if cleaning)

Suggested order by **lines removed / risk** (unchanged strategy; estimates slightly tightened):

| Phase | Action | Est. Δ | Rating lift |
|---|---|---:|---:|
| P0 | Collapse `Generate` → thin test helper over `GenerateDocument` | ~220L | +0.4 |
| P0 | Delete `structure/manager.go` + dead `pdfa` helpers + orphan `compliance/verapdf/` | ~110L | +0.3 |
| P1 | Purge unused content ops / write helpers / doc methods / dead fields; drop `_ = collectFontNames` | ~200L | +0.3 |
| P1 | Drop unused model JSON fields + always-nil layout errors + dead layout helpers | ~150L | +0.2 |
| P2 | Merge dual x10 bench scripts; unexport `BuildTable`; collapse Registry; inline identity helpers | ~100L | +0.1 |
| P2 | Shared `emitStandardFont` (**wire subset on product path**) + footer via `content.Stream`; scaleCols helper | ~80L | +0.1 |
| | **Potential after full pass** | **~850–1,150L** | **→ ~7.5–7.8** |

Do **not** “fix” leanness by adding frameworks, DI containers, or interface-for-one-impl compliance profiles. Wire existing seams or delete them.

Do **not** treat `//nolint:unused` as a cleanup — delete the operators or route layout through them.

---

## Intentional ceilings (keep, label)

These are **not** debt if marked with a `// ponytail:` comment:

1. **`map[string]interface{}` PDF dicts** — dynamic Catalog/Page/Resources keys.
2. **Pre-allocated object IDs + `AddObjectAt`** — forward refs required by PDF graph.
3. **Single shared F1 font in `GenerateDocument`** — multi-font embed is a later product decision.
4. **Dual product models** (contract note vs financial template) — two real sample products.
5. **TTF parser + subsetter bulk** — PDF/A needs embed; no stdlib path. *(Wire product subset or accept full embed as a documented ceiling.)*
6. **Heuristic text width** (`0.52 * size * len`) — full metrics only when alignment quality fails.
7. **Build-tag comply/nocomply split** — thin A/B harness for phase-8 benches.
8. **Document→P UA stub** — honest minimum until real table/sect tags land (then wire `Manager` or delete the façade permanently).
9. **Zero third-party deps** — non-negotiable ceiling for this codebase’s identity.

---

## Progress since 2026-07-25

### What improved

| Change | Evidence | Leanness impact |
|---|---|---|
| **Shared `createCatalogDict`** | Both assemblers call one helper [`engine/engine.go:318`] [`engine/document.go:223`] | Small duplication win (~20–40L avoided clone drift) |
| **`fontChain.resourceLabel` deleted** | No longer on struct [`engine/document.go:43-45`] | Tiny dead-API win (FIXED checklist item) |
| **`GenerateDocument` helper extract** | `addPageContentStreams`, `addFontChain`, `addStructureTree`, `addLoadedFontChain`, `addFallbackFontChain` | Readability / gocyclo — **not** less surface; dual path remains |
| **Watermark uses Stream ops + TjCID** | Compliance/UA fix [`engine/layout/layout.go:143-171`]; activates `Tm`/`Q` | Correctness + slight operator-surface utilization |
| **golangci-lint gate** | `.golangci.yml`, `make lint` / `lint-all` | Process hygiene; **can** force future dead-code pressure |
| **`xmpPacketPrefix` gone** | Not present in `meta/xmp.go` | FIXED prior item |
| **Bench / performance work** | `efd7682` watermark/glyph path; baselines updated | Product quality; claimed ~0.66% — **not** leanness |
| **`codehound.toml`** | Root config | Tooling only |

### What did not improve (still first-order debt)

| Debt | Status |
|---|---|
| Dual `Generate` / `GenerateDocument` assembly | **STILL OPEN** (catalog only shared) |
| `structure.Manager` unwired | **STILL OPEN** (~74 LOC pure scaffolding) |
| Dead `pdfa` helpers (3/4 functions) | **STILL OPEN** |
| Product path skips `GenerateSubset` (full RawData) | **STILL OPEN** — demo subsets; product does not |
| Dead content operators | **STILL OPEN** — now with `//nolint:unused` paper |
| `_ = collectFontNames` | **WORSE form** of same dead loop |
| Unused model JSON fields | **STILL OPEN** |
| Dual x10 bench scripts | **STILL OPEN** |
| Orphan `compliance/verapdf/` | **STILL OPEN** |
| `ModeEmbedFonts` never gates | **STILL OPEN** |
| Write/doc dead methods (`WriteComment`, `WriteTrailer`, `Name`, `AddObject`, `SetTrailerInfo`, `pagesRootRef`) | **STILL OPEN** |
| Net LOC | **+~69** (6,622 → 6,691) — cleanups did not shrink the tree |

### Anti-progress / caution signals

1. **`//nolint:unused` on 11 content operators** — lint gate learned to silence dead code rather than delete it.
2. **`_ = collectFontNames(cfg.Pages)`** — preserves dead work to satisfy “used” analysis aesthetics; pure delete is cleaner.
3. **Mechanical refactor mistaken for leanness** — `document.go` / `engine.go` churn was largely gofmt + extract-for-funlen; dual architecture unchanged.
4. **Overall must not enter 8s** while dual assembly + dead Manager remain — this update stays at **6.5**.

---

## Agents used

| Role | Focus | Rating |
|---|---|---:|
| Orchestrator (this document) | Full checklist re-walk, grep call-site audit, LOC, go.mod, commit delta since 2026-07-25 | synthesis |
| Prior Subagents A/B/C | Baseline checklist (2026-07-25) | 6.4 / 6.7 / 5.7 |

No parallel explore subagents this run — prior checklist was treated as a regression matrix against the current tree.

---

## Bottom line

| | |
|---|---|
| **Overall ponytail rating** | **6.5 / 10** (was **6.3**, **Δ +0.2**) |
| Band | First-order bloat — works with known removable surface |
| Net removable (estimate) | **~1,150 lines** across ~68 checklist items (~2 FIXED, ~3 PARTIAL, rest STILL OPEN) |
| Best single move | Collapse dual `Generate` / `GenerateDocument` assembly |
| Do not add | New deps, one-impl interfaces, more schema fields “for later”, more `//nolint:unused` |

**Lean already on dependencies (10/10). Not lean yet on surface area (4.5/10 dead API).** Lint gates and micro-benches do not retire scaffolding. Ship product features by wiring or deleting seams — never by scaffolding a second parallel path, and never by silencing unused code without removing it.
