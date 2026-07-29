# gocorepdfengine — Ponytail Ultra Review (Follow-up)

> **Generated:** 2026-07-29  
> **Mode:** Ultra (maximum aggression — find everything questionable)  
> **Baseline:** [ponytail-ultra-2026-07-25.md](./ponytail-ultra-2026-07-25.md) — **6.3/10**  
> **Method:** Ponytail skill + 3 parallel explore subagents (core · features · compliance)  
> **Scope:** `engine/**`, `sampledata/**`, compliance harness (high level)  
> **Overall rating:** **6.8 / 10** (**+0.5**)  
> **Product throughput (side):** **8.5 / 10** — not a leanness axis  
> **Total Go under review:** **~8,300 LOC** (engine + sampledata, incl. tests)  
> **go.mod deps:** **still zero**

---

## Performance context (leanness vs speed)

Ponytail scores **removable surface**, not ops/sec. Still, the compliant bench ladder matters as evidence of **intentional hot-path ceilings** (pools, image LRU, pre-parse, subset on product path):

| Snapshot | Mean ops/sec | Best |
|----------|-------------:|-----:|
| b4 (`baselines/b4_zerodha_bench_x10/`) | **1155.75** | 1226 |
| 2k mid | **1947.97** | 2033 |
| Current latest | **2349.29** | **2530** |
| Peak historical (pruned dir) | **2739** | **2941** |

**~2.0× mean** vs b4 is real product value. That raises **intentional shortcuts** and slightly lifts overall — it does **not** delete dual assembly or `structure.Manager`.

---

## Overall Rating

| Metric | Score | Δ vs 07-25 | Notes |
|---|---:|---:|---|
| **Ponytail leanness (overall)** | **6.8 / 10** | **+0.5** | Still first-order bloat; subset + hot-path ceilings credited |
| Dead API surface | **4.6 / 10** | +0.3 | Content ops trimmed; Manager/pdfa/write still dead |
| Duplication | **5.3 / 10** | +0.3 | Helpers extracted; dual assembly + dual font emit remain |
| Over-abstraction | **5.4 / 10** | +0.1 | Registry / dual flags / unwired Manager unchanged |
| Intentional shortcuts | **8.6 / 10** | +0.6 | F1 + width heuristic + subset path + pools/LRU/pre-parse that bought **~2×** |
| Dependency bloat | **10 / 10** | 0 | Zero third-party Go deps |
| Scaffolding debt | **3.6 / 10** | +0.1 | Manager, tag zoo, orphan verapdf, forward-looking UA gate |

### Rating by area

| # | Area | Rating | Open items | ~Slop | Verdict |
|---|---|---:|---:|---:|---|
| 1 | Core assembly + encode | **6.8** | 20 | ~420L | Dual assemblers remain; pools/helpers/hot path pay off in ops/sec |
| 2 | Feature modules (font/layout/image/render/model) | **7.1** | 26 | ~380L | Subset + PERF on product path; schema theatre remains |
| 3 | Compliance + samples + harness | **6.0** | 17 | ~380L | Live harness + baselines prove capacity; Manager/pdfa still scaffolding |
| | **Weighted overall** | **6.8** | **~63** | **~1,180L** | |

Weights: Area 1 25% · Area 2 50% · Area 3 25% (by relative code weight).

```
6.8×0.25 + 7.1×0.50 + 6.0×0.25
= 1.700 + 3.550 + 1.500
= 6.750 → 6.8
```

Previous: `6.4×0.25 + 6.7×0.50 + 5.7×0.25 = 6.375 → 6.3`

### Rating scale

| Score | Meaning |
|---|---|
| 9.0–10 | YAGNI-clean; only documented `ponytail:` ceilings remain |
| 8.0–8.9 | Lean for capability; localized third-order slop |
| 7.0–7.9 | Second-order slop: dead exports, duplicated helpers, thin wrappers |
| 6.0–6.9 | **← current (6.8)** First-order surface bloat; strong hot-path speed |
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

gocorepdfengine remains a **stdlib-only** PDF 2.0 / A-4 / minimal-UA engine with a real product path (`model` → `render` → `GenerateDocument`). Since 2026-07-25, **subsetting is on the product path** and format-4 cmap construction is no longer a silent no-op — that is product value, not slop.

Ponytail still sees **~1,180 lines of first-order removable surface**:

1. **Dual assembly** — `Generate` (~300 LOC smoke) still rebuilds the graph that production never uses.
2. **Dead API** — content method leftovers, write helpers, doc methods, layout helpers, test-only font exports.
3. **Compliance scaffolding** — `structure.Manager` 100% unwired; 3 of 4 `pdfa` helpers unused; Document→P only.
4. **Schema theatre** — many model JSON fields still accepted then ignored.

**Bottom line: 6.8/10 (+0.5).** Speed is real (~2×); leanness is still mid-band. Best single move remains collapse `Generate` → thin wrapper over `GenerateDocument` **without** regressing baselines. Do not add deps to “fix” leanness.

---

## Delta vs 2026-07-25

### RESOLVED

| Item | Note |
|---|---|
| Product-path font subset | `setupDocumentFont` runs `GenerateSubset` before embed (`document.go:316-320`); smoke path also subsets |
| `glyphIDArray` / `idRangeOffset` forced-zero dead path | Format-4 cmap now builds real glyphIDArray (`subset.go:675-716`) |
| Dead `xmpPacketPrefix` | Gone from `meta/xmp.go` |
| Several dead content operators as methods | Removed: `re`/`m`/`l`/`h`/`f`/`s`/`w`/`j`/`rg`/`cm`/`q` as `Stream` methods |

### PARTIAL

| Item | Status |
|---|---|
| Dual assembly `Generate` ≈ `GenerateDocument` | Still two full assemblers; helpers only on multi-page path |
| Catalog / XMP / ICC duplication | Shared only on multi-page path; smoke path still inlines |
| Dead content surface | Smaller set remains: `Tm`, `TL`, `S`, `J`, `RG`, `Q`, `BMC`, `ArtifactBMC`, `Compress` |
| Font A-4 fallback + glyph alphabet | Still duplicated in `addFontObjects` and `setupDocumentFont` |

### STILL OPEN

Essentially the rest of the 2026-07-25 checklist: dead `Result`, `fontNames`/`UsedFonts`/`resourceLabel`, `ModeEmbedFonts` never checked, `AddObject`/`SetTrailerInfo`/`SetPagesRoot`, write helpers, layout `WrapText`/`Point`/`MCID`/`ImageObj.Data`, font test-only exports, `Registry`, ignored model JSON fields, entire `structure.Manager`, dead `pdfa` helpers, tag zoo, dual bench scripts, orphan `compliance/verapdf/`.

### NEW debt

| Tag | Item |
|---|---|
| `shrink:` | **Near-clone font emitters:** `addFontObjects` vs `setupDocumentFont` |
| `shrink:` | **Codehound const / ignore inflation** (~net +1.6k LOC vs prior ~6.6k estimate) |
| `yagni:` | **Triple zlib/flate pools** — engine, content (dead), image + color |
| `yagni:` | Content methods unused while layout hand-emits same operators |

---

## What was reviewed

| Area | Paths | Agent |
|------|-------|-------|
| 1 Core assembly + encode | `engine/engine.go`, `document.go`, `doc/`, `write/`, `page/`, `content/` | Subagent A |
| 2 Feature modules | `engine/font/`, `layout/`, `image/`, `render/`, `model/` | Subagent B |
| 3 Compliance + harness | `engine/pdfa/`, `structure/`, `meta/`, `color/`, `sampledata/`, `compliance/` | Subagent C |

---

## 1. Core assembly + encode — Rating **6.8 / 10**

**Targets:** core files | **~420 lines removable** | **Open items:** 20

### Checklist

- [ ] `yagni:` **Dual assembly path.** `Generate` still full parallel assembler; only tests call it. Collapse to thin smoke over `GenerateDocument`. [`engine/engine.go:94-303`] vs [`engine/document.go:63-231`]
- [ ] `delete:` **`engine.Result`.** One-field `{Data []byte}`. [`engine/engine.go:90-92`]
- [ ] `delete:` **Dead `fontNames` collection.** Built, never used for emission. [`engine/document.go:96-108`]
- [ ] `delete:` **`PageContent.UsedFonts`.** Only feeds dead `fontNames`. [`engine/document.go:36,99-100`]
- [ ] `delete:` **`fontChain.resourceLabel`.** Set `"F1"`, never read. [`engine/document.go:40-42,117-118`]
- [ ] `delete:` **`doc.ModeEmbedFonts`.** OR’d in assemblers/render; `HasMode(ModeEmbedFonts)` has **zero** callers. [`engine/doc/builder.go:28`]
- [ ] `delete:` **`Document.AddObject`.** Only `AddObjectAt` used. [`engine/doc/builder.go:70-75`]
- [ ] `delete:` **`Document.SetTrailerInfo`.** Zero callers. [`engine/doc/builder.go:95-98`]
- [ ] `delete:` **`SetPagesRoot` / `pagesRootRef`.** Written, never read in `Build`. [`engine/doc/builder.go:46-47,90-93`]
- [ ] `delete:` **`Build` `[]byte` / `default` branches.** No live non-dict/stream objects. [`engine/doc/builder.go:154-160`]
- [ ] `delete:` **`write.WriteComment`.** Zero callers. [`engine/write/writer.go:76-79`]
- [ ] `delete:` **`write.WriteTrailer`.** Dead; `Build` hand-rolls trailer. [`engine/write/writer.go:165-177`]
- [ ] `delete:` **`write.Name`.** Defined, never referenced. [`engine/write/writer.go:194-197`]
- [ ] `delete:` **Unused `content.Stream` methods.** Never called: `Tm`, `TL`, `S`, `J`, `RG`, `Q`, `BMC`, `ArtifactBMC`. [`engine/content/content.go:117-176`]
- [ ] `delete:` **`content.Stream.Compress` + `Compressed`.** Zero callers. [`engine/content/content.go:41,183-208`]
- [ ] `shrink:` **Footer/page-number emission.** Manual hex CID loops vs `content.Stream`. [`engine/document.go:233-290`]
- [ ] `shrink:` **Duplicated PDF string escape.** `content.Tj` ≈ `write.StringLit`.
- [ ] `shrink:` **NEW — dual A-4 font emit.** `addFontObjects` ≈ `setupDocumentFont`. [`engine.go:305-389`] / [`document.go:292-354`]
- [ ] `yagni:` **`Page.FontResources` / `XObjectResources` dual storage.** Assigned then `ToDict` takes separate maps.
- [ ] `yagni:` **`page` package is a pure dict serializer.** Keep only as ceiling.
- [ ] `yagni:` **Empty-page default.** `len(cfg.Pages)==0` injects blank stream.
- [x] `ponytail:` **`map[string]interface{}` PDF dicts** — dynamic keys.
- [x] `ponytail:` **Pre-allocate IDs + `AddObjectAt`** — forward refs.
- [x] `ponytail:` **Single shared F1 font (v1)** in `GenerateDocument`.
- [x] `ponytail:` **`write.Encoder` as byte sink**.
- [x] **PARTIAL:** Catalog helpers on multi-page path only.
- [x] **PARTIAL:** Content path ops reduced (many path ops no longer exist as methods).

#### Scorecard notes

| Dimension | Score |
|---|---:|
| Dead API surface | 4.2 |
| Duplication | 4.5 |
| Over-abstraction | 5.5 |
| Intentional shortcuts | 8.2 |

---

## 2. Feature modules — Rating **7.1 / 10**

**Targets:** 18 files | **~380 lines removable** | **Open items:** 26

### Checklist

#### Layout

- [ ] `delete:` **`WrapText`** — zero callers. [`layout/layout.go:91-136`]
- [ ] `delete:` **`Point` type** unused. [`layout/layout.go:24-27`]
- [ ] `delete:` **`ContentBuilder.MCID`** written, never read.
- [ ] `delete:` **`ImageObj.Data`** duplicates `Img.Data`.
- [ ] `delete:` **No-op color re-assign.** [`layout/table.go:195-197`]
- [ ] `delete:` **`LayOut`/`LayOutFrom` always `nil` error** — drop `error`.
- [ ] `yagni:` **`contentBottom` always `= marginTop`.**
- [ ] `delete:` **`CellProps.Bold`/`Italic`/`Underline`** parsed, never applied.
- [x] `yagni:` **`theme.go` one-caller helpers** — fine as thin helpers.
- [x] `ponytail:` **`textWidth = len*size*0.52`.**

#### Font

- [ ] `delete:` **`Widths` / `CharWidth` / `GlyphCount`** — production-dead (tests only).
- [ ] `delete:` **`IsTTF`** — zero callers.
- [ ] `delete:` **`AddChars`** — prod uses `AddChar` only.
- [ ] `delete:` **`UsedChars`** alias of `UsedRunes`.
- [ ] `delete:` **`Family` / `IsSerif` / `IsMono`** set, never read.
- [ ] `delete:` **`parseHMTX` / `parseMaxp`** existence-check only.
- [ ] `delete:` **`tableDir` entries + `requiredTags`** built then discarded.
- [x] `delete:` ~~**glyphIDArray with idRangeOffset forced 0**~~ — **FIXED**
- [ ] `yagni:` **`Registry`** not multi-font in practice — collapse to `LoadStandardFont`.
- [ ] `shrink:` **`CIDSystemInfoDict`** always `"Adobe","Identity",0`.
- [x] `ponytail:` **TTF load + subset bulk** — required for PDF/A.
- [x] **RESOLVED:** Product path subsets fonts. [`document.go:316-320`]

#### Image

- [ ] `delete:` **`image.XObjectDict`** unused in prod (tests only).
- [ ] `yagni:` **Process-global SHA-256 image cache.**
- [ ] `yagni:` **PNG→JPEG for `w*h >= 100*100`** — undocumented, no opt-out.
- [ ] `stdlib:` **JPEG SOF walk** vs `image.DecodeConfig` for dimensions.

#### Render / model

- [ ] `shrink:` **`scaleCols` ≈ `scaleColsWeights`.**
- [ ] `shrink:` **Page-assembly loops** duplicated in `PDF` / `TemplatePDF`.
- [ ] `delete:` **`pageDimensions`** one-line pass-through.
- [ ] `yagni:` **`TemplatePDF(..., opts Options)` ignores `opts`** (`_` param).
- [ ] `yagni:` **`PDFACompliant` OR `ArlingtonCompatible`** → same mode bits.
- [ ] `delete:` **Ignored model JSON fields:** `Config.PageBorder`/`PageAlignment`/`EmbedFonts`; `Title.Link`; `TableCell.Link`/`Dest`; `Image.ImageName`; `Compliance`/`Features.Bookmarks`/`InternalLinks`; `Metadata.Keywords`; `Trade.Currency`; `Footer.Font`/`Link`; `Audit.AuditorSignaturePlaceholder`.
- [ ] `yagni:` **`BuildTable` exported**, only used inside package.
- [ ] `shrink:` **`buildRetail` / `buildActive` / `buildHFT`** share patterns.
- [x] `ponytail:` **Dual product surfaces** (contract note vs template). Keep until products merge.

#### Scorecard notes

| Dimension | Score |
|---|---:|
| Dead API surface | 5.5 |
| Duplication | 6.2 |
| Over-abstraction | 6.5 |
| Intentional shortcuts | 8.5 |

---

## 3. Compliance + samples + harness — Rating **6.0 / 10**

**Targets:** ~16 primary files (+ compliance scripts) | **~380 lines removable** | **Open items:** 17

### Package truth (live vs scaffolding)

| Package / area | Reality (2026-07-29) |
|---|---|
| `engine/meta` | **Live** — XMP from both assemblers |
| `engine/color` profiles + hex | **Live** for A-4 / theme (ThemeLink dead) |
| `engine/pdfa` | **Mostly dead** — only `OutputIntentDict` has callers |
| `engine/structure` dict helpers | **Live** for Document→P hand-assembly |
| `engine/structure.Manager` | **100% dead scaffolding** — zero external call sites |
| Font subset on product path | **Live** (was test/smoke-only relative to multi-page) |
| `sampledata/zerodha` | **Live harness**; dual x10 scripts |
| `compliance/` (root) | **Live** Makefile path |
| `compliance/verapdf/` | **Orphan** — superseded by root scripts |

### Checklist

- [ ] `delete:` **Entire `structure.Manager`** — zero external call sites. [`structure/manager.go`]
- [ ] `delete:` **`pdfa.CatalogA4Extras` / `PageResourceA4Extras` / `ImageColorSpace`** — zero Go callers.
- [ ] `yagni:` **16 of 18 `StructType` constants** unused outside `structure.go` (live: `TypeDocument`, `TypeP` only).
- [ ] `delete:` **`NamespaceRef(id)`** exported, never called.
- [ ] `yagni:` **`StructParentsValue`** identity — inline.
- [ ] `delete:` **`OBJR` type + kid branch** never constructed by callers.
- [ ] `yagni:` **`ParentTreeDict` `annots` always `nil`.**
- [ ] `yagni:` **`StructElem.Title` / `Alt`** never populated by assemblers.
- [x] `delete:` ~~**`xmpPacketPrefix`**~~ — **gone**
- [ ] `yagni:` **`XMPConfig.Keywords`** never set by callers.
- [ ] `delete:` **`color.ThemeLink`** defined, never used outside `hex.go`.
- [ ] `shrink:` **Hand-rolled ICC codec** fallback when system ICC missing. Prefer `//go:embed`.
- [ ] `yagni:` **Zerodha palette in core `engine/color`** — only contract_note consumes.
- [ ] `delete:` **Orphan `compliance/verapdf/`** — Makefile uses root scripts only.
- [ ] `shrink:` **`run_bench_x10.sh` ≈ `run_bench_x10_nocomply.sh`.**
- [ ] `stdlib:` **`simpleRNG` xorshift** vs `math/rand`.
- [ ] `yagni:` **Makefile aliases** `bench-zerodha-cached`/`uncached`/`bench-help`.
- [ ] `yagni:` **`structure_tree_check.py`** validates TR/TD/empty Sect shapes this engine never emits.
- [x] `ponytail:` Build-tag `main.go` / `main_nocomply.go` + shared `bench.go`
- [x] `ponytail:` Live compliance path: `OutputIntentDict` + ICC + XMP + Document→P

#### Scorecard notes

| Dimension | Score |
|---|---:|
| Dead API surface | 3.2 |
| Duplication | 5.2 |
| Over-abstraction | 4.2 |
| Intentional shortcuts | 7.2 |
| Scaffolding debt | 3.2 |

---

## Priority remediation (if cleaning)

| Phase | Action | Est. Δ LOC | Rating lift |
|---|---|---:|---:|
| P0 | Collapse `Generate` → thin test helper over `GenerateDocument` (+ merge dual font emit) | ~280L | +0.4 |
| P0 | Delete `structure/manager.go` + dead `pdfa` helpers + orphan `compliance/verapdf/` | ~120L | +0.3 |
| P1 | Purge unused content methods / write helpers / doc methods / dead fields | ~180L | +0.3 |
| P1 | Drop unused model JSON fields + always-nil layout errors + dead layout helpers | ~150L | +0.2 |
| P2 | Merge dual x10 scripts; unexport `BuildTable`; collapse Registry; inline identity helpers | ~100L | +0.1 |
| P2 | Footer via `content.Stream`; shared `scaleCols`; honor or drop `TemplatePDF` opts | ~80L | +0.1 |
| | **Potential after full pass** | **~850–1,000L** | **→ ~7.6–7.9** |

Do **not** add frameworks/DI/compliance interfaces-for-one-impl. Wire or delete.

---

## Intentional ceilings (keep, label)

These are **not** debt if marked with a `// ponytail:` comment:

1. **`map[string]interface{}` PDF dicts** — dynamic Catalog/Page/Resources keys.
2. **Pre-allocated object IDs + `AddObjectAt`** — PDF forward refs.
3. **Single shared F1 font in `GenerateDocument`** — multi-font is a later product decision.
4. **Dual product models** (contract note vs financial template).
5. **TTF parser + subsetter bulk** — PDF/A embed; no stdlib path.
6. **Heuristic text width (`0.52`)** — full metrics only if alignment quality fails.
7. **Build-tag comply/nocomply split** — thin A/B bench harness.
8. **Document→P UA stub** — honest minimum until real table/sect tags land (then wire `Manager` or delete it permanently).
9. **NEW ceiling candidate:** global image cache for bench throughput — document as intentional if multi-doc cache is desired; else delete.

---

## Agents used

| Agent | Focus | Rating |
|---|---|---:|
| Subagent A (explore) | Core assembly + encode | 6.8 |
| Subagent B (explore) | Font / layout / image / render / model | 7.1 |
| Subagent C (explore) | Compliance + samples + harness | 6.0 |

Synthesis + weighted overall by orchestrator; **perf correction** after reading `baselines/` (~2× ops/sec).

---

## Bottom line

| | |
|---|---|
| **Overall ponytail rating** | **6.8 / 10** (+0.5 vs 6.3) |
| **Product throughput (side)** | **8.5 / 10** (~2.3–2.5k ops/sec compliant; peak hist. ~2.9k) |
| Band | Still **first-order bloat** on surface — **strong** on hot-path product speed |
| Net removable (estimate) | **~1,180 lines** across ~63 open checklist items |
| Best single move | Collapse dual assembly **while keeping baseline gates green** |
| Biggest real wins since 07-25 | **~2× throughput** + product-path subset + format-4 cmap |
| Do not add | New deps, one-impl interfaces, more schema fields “for later” |
| Go LOC under review | **~8,300** (engine+sampledata incl. tests) |
| **go.mod third-party deps** | **0** (score **10/10**) |

**Lean on dependencies. Fast on product path. Not lean on surface area (dead API 4.6/10).** Ship by wiring or deleting seams — never by scaffolding a second path that risks the 2× win.
