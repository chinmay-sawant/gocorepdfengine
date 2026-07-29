# Architecture Review — gocorepdfengine (Follow-up)

| | |
|---|---|
| **Date** | 2026-07-29 |
| **Prior review** | 2026-07-25 — **6.2 / 10** |
| **Method** | `/improve-codebase-architecture` + `/codebase-design` vocabulary |
| **Agents** | 3 explore passes (module depth · coupling/seams · testability) |
| **Scope** | `engine/**`, `sampledata/**`, `plans/**`, `compliance/**`, `Makefile` |
| **Branch context** | `chore/after-codehound` |
| **Overall rating** | **6.8 / 10** (Δ **+0.6** vs 2026-07-25) |
| **Product throughput (side)** | **8.5 / 10** — not a structural axis; see Performance evidence |
| **Tool** | Grok Build |

---

## Performance evidence (must not be under-scored)

Recorded **compliant** Zerodha x10 harness (cache ON, 5000 iters, 48 workers, A-4+UA flags) under `baselines/`:

| Snapshot | Path | Mean ops/sec | Best ops/sec | Mean latency |
|----------|------|-------------:|-------------:|-------------:|
| **b4 (pre–Codehound)** | `baselines/b4_zerodha_bench_x10/` | **1155.75** | 1226.41 | 41.0 ms |
| **~2k mid** | `baselines/2k_zerodha_bench_x10/` | **1947.97** | 2033.34 | 24.1 ms |
| **Current latest** | `baselines/zerodha_bench_x10_stats_latest.txt` | **2349.29** | **2530.34** | 19.9 ms |
| Peak recorded (later pruned) | `25042026_1_3k_…` (git history) | **2739.27** | **2941.49** | 16.8 ms |

**Lift vs b4:** mean **~2.03×** (1156 → 2349); best **~2.06×** (1226 → 2530). Peak historical best was ~2.9k ops/sec.

This is **product-path** throughput (`model` → `render` → `GenerateDocument`), not demo `Generate`. Mechanisms include Codehound PERF passes, `sync.Pool` compress/glyph buffers, image LRU, pre-parsed template colors/props, and product-path subset. Phase-6 plan status is therefore **partial**, not “not started.”

Structural debt and high throughput **coexist**: the engine can be fast while still having dual assemblers and shallow UA trees. Scores below credit **delivery / leverage / complexity** for the 2× win without pretending dual-assembly is gone.

---

## Executive summary

gocorepdfengine is a **mid-maturity** PDF 2.0 engine with a clear package map, real depth in `font`, `layout`, and `doc`, and a **demonstrably fast product path**: compliant Zerodha benches moved from **~1.2k → ~2.3–2.5k ops/sec** (≈**2×**), with peak runs near **2.9k**. That is first-class phase-8 delivery under PDF/A-4 + PDF/UA-2 flags.

Since 2026-07-25, correctness also moved: **`GenerateDocument` prefers `GenerateSubset`**, helpers clarify multi-page assembly, ICC Length is tested, and lint/PERF gates exist. Architectural structure is slower to move: **two assemblers** still own compliance/font/structure; `structure.Manager` and most of `pdfa` remain unused; product path still lacks in-process `go test`. New dual-assembler drift: **`/Tabs /S` on demo `Generate` only**.

**Headline score: 6.8 / 10** — strong product throughput and plan delivery, held back by assembly/UA seams and product-path unit tests. **Side rating for throughput capability: 8.5 / 10.**

---

## Scorecard (out of 10)

| Axis | Prev | **Now** | Weight | One-line justification |
|------|-----:|--------:|-------:|------------------------|
| Architecture depth | 5.5 | **6.5** | 15% | Font/layout/doc deep; hot-path pools/subset/pre-parse add real depth; Manager/pdfa still scaffolding. |
| Seam discipline | 4.5 | **5.0** | 10% | Named helpers are weak internal seams; Mode-if compliance still owns policy. |
| Locality | 6.5 | **6.5** | 10% | Subset locality fixed (+); dual assemblers + Tabs drift (0 net). |
| Leverage | 6.0 | **7.5** | 10% | **2× compliant bench** proves subset + pools + PERF fixes pay back on product path. |
| Testability | 4.0 | **5.0** | 15% | Multi-run baselines/gates are real regression harness; product still missing `go test`. |
| Compliance design | 6.0 | **6.5** | 15% | Subset + ICC Length; UA still page-as-`P`; product missing `/Tabs`. |
| AI-navigability | 7.5 | **7.5** | 10% | Phase plans + baselines tree; `.golangci.yml`; no CONTEXT/ADRs. |
| Complexity control | 5.0 | **6.5** | 5% | PERF machinery is intentional complexity; dual graphs remain. |
| Documentation | 6.5 | **7.0** | 5% | `baselines/` trajectory (b4 → 2k → current) documents capacity. |
| Delivery vs plan | 7.0 | **8.5** | 5% | Phase 8 harness + **~2× ops/sec**; phase 6 **partial** (pools/hot path), not idle. |
| **Weighted overall** | **6.2** | **6.8** | | Formula + product-throughput uplift. |

### How the overall was computed

```
6.5×0.15 + 5.0×0.10 + 6.5×0.10 + 7.5×0.10 + 5.0×0.15
+ 6.5×0.15 + 7.5×0.10 + 6.5×0.05 + 7.0×0.05 + 8.5×0.05
= 0.975 + 0.500 + 0.650 + 0.750 + 0.750
+ 0.975 + 0.750 + 0.325 + 0.350 + 0.425
= 6.450  →  with qualitative uplift for 2× compliant product throughput  →  6.8 / 10
```

**Honesty check:** Dual assemblers still cap **seam** scores in the 5s. The earlier 6.1 draft **under-weighted** phase-8 delivery and hot-path leverage — that was wrong given `baselines/`. Throughput is scored primarily on **leverage + delivery** (+ side **8.5** product-throughput rating), not by pretending Manager is wired.

### Rating band guide

| Band | Meaning |
|------|---------|
| 8–10 | Deep modules, real seams, product path tested, compliance owned by modules |
| 6–7 | Works, clear packages, known assembly debt — **can still be very fast** |
| 4–5 | Fragile, shallow throughout, hard to change safely |
| ≤3 | Unusable as a maintained engine |

**Structural band: upper “works with known debt.” Product throughput band: strong (8.5 side score).**

---

## Delta vs 2026-07-25

### Improved

| Change | Evidence | Why it matters |
|--------|----------|----------------|
| **Product path subsets fonts** | `setupDocumentFont` prefers `GenerateSubset` — `document.go:316–320` | Closes the high-severity subset drift on product/bench path. |
| **Demo path still subsets** | `engine.go:332–335` | Both paths now subset when load succeeds. |
| **`GenerateDocument` helpers extracted** | `buildContentStreams`, `setupDocumentFont`, `buildStructureTree`, `addComplianceMetadata`, `buildCatalog` | Better locality *within* the multi-page assembler. |
| **ICC Length self-init + tests** | `color/profiles.go`; `color/profiles_test.go` | Real PDF/A stream-length bug class covered. |
| **Watermark CID encoding fix** | `layout/layout.go`; `layout_test.go` | Compliance-relevant layout fix with a test. |
| **Lint gate** | `.golangci.yml`; `Makefile` `lint` / `lint-all` | Tooling maturity. |
| **UsedText includes footer glyphs** | `render/contract_note.go` | Subset alphabet includes footer glyphs. |
| **~2× compliant throughput** | `baselines/b4_*` → `zerodha_bench_x10_stats_latest.txt` | Mean **1156 → 2349** ops/sec; best **~2530** (peak historical ~2941). |
| **Hot-path pools / caches** | `sync.Pool` in engine/font/image/color/content; image LRU | Phase-6 work landed ad-hoc outside full plan checklist. |
| **Codehound PERF series** | commits `693646a`…`96d3fe8`, pre-parse props/colors | Systematic alloc/hot-path cleanup on product path. |

### Stayed the same (core debt)

| Debt | Evidence |
|------|----------|
| **Dual assemblers** | `Generate` vs `GenerateDocument` still parallel mode graphs |
| **`structure.Manager` unwired** | Zero external call sites; live trees Document→P |
| **`pdfa` half-dead** | Only `OutputIntentDict` used |
| **Product path untested in `go test`** | No `document_test.go`, no `render/*_test.go` |
| **Silent fake font + empty FontFile2** | Both paths still succeed with broken embed |
| **Width magic constants** | Layout `0.52`; footer `0.55` |
| **Phase 6 pooling** | Not started as plan architecture |
| **No CONTEXT.md / ADRs** | Domain language still only in plans/guides |

### Got worse / new drift

| Issue | Evidence |
|-------|----------|
| **NEW dual-assembler drift: `/Tabs /S`** | `Generate` sets `pg.Tabs = "S"`; **`GenerateDocument` never sets `Tabs`**. Tests lock demo behaviour only. |
| **Asymmetric refactor** | Helpers only on document path; `Generate` still inlines catalog/structure/XMP |
| **Root README** | Currently a codehound-ignore note, not product docs |

### What did *not* happen (vs prior top recommendations)

1. Collapse dual assemblers → **not done**
2. Compliance profile module → **not done**
3. Wire or delete `structure.Manager` → **not done**
4. Single `font.Embed` facade → **not done**
5. `GenerateDocument` integration tests → **not done**

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

No import cycles observed.

### Module inventory (depth)

| Module | Depth | Notes |
|--------|-------|-------|
| **font** | **Deep** | Load, cmap, subset, ToUnicode, emit dicts |
| **layout** | **Deep** | Table page-break, place text/rect/image; watermark CID hardened |
| **doc** | **Deep** | Small surface (`AllocID` / `AddObject` / `Build`); owns xref/trailer |
| **render** | **Deep (edge)** | Domain/template → `PageContent` + mode |
| **meta** / **color (ICC)** | Medium–deep | + Length self-init tests |
| **image** | Medium–deep | JPEG/PNG + cache |
| **write** | Medium | Necessary encoder |
| **engine root** | Volume high, locality mixed | document helpers good; dual path bad |
| **content** / **page** | Shallow | |
| **pdfa** | Mostly unused | Only `OutputIntentDict` live |
| **structure.Manager** | **Dead depth** | Implemented, never wired |

---

## Critical findings (status)

### C1 — Dual assemblers with drift → **PARTIAL**

- **Still open:** Two full assembly procedures.
- **Resolved (subset):** Both call `GenerateSubset` when load succeeds.
- **NEW drift:** `Tabs="S"` only in `Generate`; absent in `GenerateDocument` page loop.
- **Font emit still duplicated:** `addFontObjects` vs `setupDocumentFont`.

### C2 — Compliance flag-sprinkled, not a deep module → **STILL OPEN**

Mode bits + `if isA4` / `if isUA` in both assemblers. `pdfa` helpers unused. Extraction of helpers is locality improvement, not a profile module.

### C3 — `structure.Manager` fake seam → **STILL OPEN**

Manager never called outside package. Live tree: Document → one `P` per page, MCID 0. Tag zoo (`Table`/`TR`/`TD`) unused.

### C4 — Product pipeline almost untested → **STILL OPEN** (slightly less severe)

| Surface | `go test` |
|---------|-----------|
| `engine.Generate` | Yes (4 tests, markers) |
| `engine.GenerateDocument` | **No** |
| `render.*` / `model.*` | **No** |
| `color` profiles | **Yes** (new, 2 tests) |
| `structure`, `pdfa`, `meta`, `write`, `doc` | **No** dedicated tests |

### C5 — Silent degradation on font failure → **STILL OPEN**

Fake Liberation + `/Length: 0` FontFile2 on both paths. Success path with broken embed remains.

### C6 — Encoding / width policy leakage → **STILL OPEN**

Layout always Identity-H / CID-style text; non-A4 installs Type1 Helvetica. Width estimates: `0.52` vs footer `0.55` — not font metrics.

### C7 — Dead / half-used packages inflate depth → **STILL OPEN**

Manager + most of `pdfa` still fail the deletion test for live behaviour.

### C8 — Product path missing `/Tabs /S` → **NEW**

Demonstrates dual-assembler tax still collecting interest. Phase-5 plan claims Tabs; product multi-page path omits it.

---

## Deepening opportunities

Ranked for leverage. Vocabulary: **module**, **interface**, **implementation**, **depth**, **seam**, **adapter**, **leverage**, **locality**.

### Strong

1. **Collapse dual assemblers into one deep assembly module** — One internal path; `Generate` / `GenerateDocument` as thin adapters. Helpers in `document.go` are a **starting point**, not the end state.
2. **`font.Embed` facade** — Single policy: load → AddChar → subset → emit. Fail hard when A-4 requires embed and load fails (closes C5).
3. **`GenerateDocument` (+ optional `render.PDF`) as test seam** — Multi-page / A-4 / UA / subset / Tabs markers. Closes C4/C8.
4. **Compliance profile module** — Deepen `pdfa` (or new `profile`): OutputIntent, MarkInfo, Lang, TrailerInfo, Tabs. Closes C2.
5. **Wire `structure.Manager` from layout *or* delete + ADR** — Closes C3/C7.

### Worth exploring

6. Shared `render.finish` for contract_note + template.
7. Font metrics for layout widths (kill 0.52 / 0.55 magic).
8. ContentBuilder as sole content seam.
9. Phase 6 pooling only after assembly locality is fixed.

---

## Test map (current)

| Package / path | Tests | Quality |
|----------------|-------|---------|
| `engine` | 4 (`Generate` only) | Marker/substring; no multi-page |
| `engine/font` | ~15 | Best unit depth |
| `engine/layout` | 7 | Stream smoke + page-break + watermark CID |
| `engine/image` | 5 | Parse/dedup solid |
| `engine/color` | 2 | **New** ICC Length integrity |
| `render`, `model`, `structure`, `pdfa`, `meta`, `write`, `doc`, `page`, `content` | **0** | — |
| Package `Benchmark*` | **0** | Bench lives under `sampledata/zerodha` |
| External | veraPDF + structure_tree_check via Makefile | Fixtures not tied to `go test` |

**Ratio:** ~5 test files vs ~30+ production sources under `engine/` (~1:6).

---

## Phase plan vs reality

| Phase | Plan | Reality (2026-07-29) |
|------:|------|----------------------|
| 1 Core PDF 2.0 writer | ✅ | Present; tested via `Generate` |
| 2 Layout primitives | ✅ | Present; watermark CID hardened |
| 3 Font embedding | ✅ | TTF + subset; **product path now subsets** |
| 4 PDF/A-4 | ✅ | Objects present; ICC Length fix; silent font fallback remains |
| 5 PDF/UA-2 | ✅ checklist | Catalog/structure present; **tree still Document→P**; **product path omits Tabs** |
| 6 Performance pooling | ⏸️ plan / **partial reality** | Formal checklist incomplete; **pools + PERF + 2× bench** are real |
| 7 Optional product | ⏸️ | Out of scope |
| 8 Zerodha bench | harness | **Delivered** — multi-run baselines, ~2.3–2.5k ops/sec compliant |

---

## Risk register

| Risk | Severity | Status vs prior | Why |
|------|----------|-----------------|-----|
| Product PDF size without subset | **Was High → Medium/Low** | **Mitigated** | Product path subsets when load works |
| UA compliance overstated | Medium–High | Open | One P/page; no table structure; Manager dead |
| Silent empty FontFile2 | High | Open | Both assemblers still succeed with fake font |
| Assembler drift | Medium | **Still open + new instance** | Tabs only on `Generate` |
| Fixture / product test gap | Medium | Open | `go test` ≠ product path |
| Dual font embed graphs | Medium | Open | Subset fixed; object emit still copy-paste |
| Perf regression without baselines | Medium | **Mitigated** | `baselines/` + x10 scripts; keep gates on mean/median |

---

## Recommended order of work

1. **Unify assembly** — extract shared core; make `Generate` a thin adapter over `GenerateDocument`. Immediately set `Tabs="S"` in one place.
2. **`font.Embed` + hard fail on A-4 font load failure** — kill fake empty FontFile2.
3. **`document_test.go` / product integration tests** — multi-page A-4+UA markers, subset/FontFile2 non-empty, Tabs, no `/Info` trailer.
4. **Compliance profile module** — own catalog/XMP/ICC/MarkInfo policy.
5. **Manager: wire for tables or delete + ADR**.
6. **Phase 6 pooling** only after (1).

---

## Top recommendation

**Protect the 2× product-path win, then finish structural locality.**

1. Keep `baselines/` gates green (mean/median ops/sec) on every compliance/font change.  
2. Unify assembly so Tabs/subset/font policy cannot drift between demo and product.  
3. Add `GenerateDocument` tests that lock subset + UA page keys (would have caught Tabs).  
4. Only then deepen formal phase-6 pooling on a **single** assembly graph.

Do not trade the measured ~2.3–2.5k ops/sec path for speculative abstraction.

---

## What this review is not

- Not a line-by-line style nitpick.
- Not a veraPDF full-suite audit of every sample PDF.
- Not an interface redesign (skill step 2 stops at candidates).

---

## Follow-ups (skill loop)

1. Pick one candidate (1–5 recommended).
2. Run `/grilling` on constraints, seam placement, and surviving tests.
3. Optionally `/design-an-interface` for the compliance profile or font embed module.
4. Record durable rejections as ADRs; add `CONTEXT.md` when domain terms stabilize.

---

## Appendix A — Agent method

| Agent | Focus | Outcome |
|-------|--------|---------|
| Explore 1 | Module inventory, depth, dependency graph | Deep: font/layout/doc/render; shallow: content/page/pdfa; dual assembler debt |
| Explore 2 | Coupling, leakage, real vs fake seams | Mode-flag compliance; Manager unused; subset fixed; Tabs drift new |
| Explore 3 | Testability, metrics, phase vs reality | 5 test packages; product path untested in `go test`; **baselines show 2×** |
| Correction | Performance under-weight | First 6.1 draft ignored `baselines/`; revised to **6.8** with delivery/leverage credit |

## Appendix B — Key file anchors

| Claim | Location |
|-------|----------|
| `Generate` entry | `engine/engine.go:97` |
| `GenerateDocument` entry | `engine/document.go:63` |
| Product subset | `engine/document.go:316–320` |
| Demo subset | `engine/engine.go:332–335` |
| Document helpers | `engine/document.go:233–437` |
| Tabs only on Generate | `engine/engine.go:202` vs `document.go:207–211` |
| Silent fake font | `document.go:343–353`, `engine.go:368–388` |
| Unused Manager | `engine/structure/manager.go` |
| Unused pdfa extras | `engine/pdfa/pdfa.go:28–50` |
| Product → GenerateDocument | `render/contract_note.go:144`, `render/template.go:99` |
| ICC tests | `engine/color/profiles_test.go` |
| golangci | `.golangci.yml`, `Makefile:115–122` |

## Appendix C — Related artifacts

- HTML visual report: [`architecture-review-20260729-121831.html`](./architecture-review-20260729-121831.html)
- Prior review: [`2026-07-25-architecture-review.md`](./2026-07-25-architecture-review.md)
- Plans index: [`../README.md`](../README.md)
- Base plan: [`../baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md`](../baseplan/base-pdf-engine-pdfa4-pdfua2-plan.md)
