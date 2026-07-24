# Phase 8 — Zerodha-Style Benchmark (gocorepdfengine only)

**Status:** Harness on local engine (JSON → model → layout → PDF)  
**No gopdfsuit dependency.** Templates only inspired by the Zerodha gold-standard mix.

---

## Goal

Benchmark gocorepdfengine’s **layout + coloring + document assembly** on a Zerodha-like 80/15/5 workload:

| Tier | Share | Source |
|------|-------|--------|
| Retail | 80% | `retail_investor.json` |
| Active | 15% | `active_trader.json` → expand **40** trades |
| HFT | 5% | `hft_algo.json` → expand **2000** trades |

Two dimensions:

1. **Compliance:** compliant (A-4 + UA-2 flags) vs non-compliant (PDF 2.0 only)  
2. **Model cache:** cached (expand once) vs non-cached (rebuild every iteration)

---

## Pipeline (local packages only)

```
sampledata/zerodha/*.json
        │
        ▼
engine/model.LoadJSON + ExpandTrades
        │
        ▼
engine/render.BuildTable  (uses engine/color theme + engine/layout)
        │
        ▼
engine/layout.TableLayout.LayOut  (multipage, fills, borders, text color)
        │
        ▼
engine.GenerateDocument  (multi-page PDF 2.0 / A-4 / UA-2)
```

---

## Checklist — already landed

- [x] Remove gopdfsuit require/replace and generate wrapper  
- [x] JSON templates under `sampledata/zerodha/`  
- [x] `engine/model` JSON → `ContractNote`  
- [x] `engine/color` hex + theme palette (header/section/buy/sell/alt rows)  
- [x] `engine/layout` table layout + `StyledCell` colors/borders  
- [x] `engine/render` contract-note builder  
- [x] `engine.GenerateDocument` multi-page assembly  
- [x] Bench harness: compliant / nocomply, `BENCH_CACHE` on/off  
- [x] Makefile: `bench-zerodha*`, `bench-zerodha-cached`, `bench-zerodha-uncached`  
- [x] x5 / x10 run scripts  

---

## Checklist — layout / coloring gaps to close

- [ ] Text wrapping inside cells (long symbols)  
- [ ] True diagonal watermark (cm rotation)  
- [ ] Per-cell borders L/R/T/B like props `1:0:0:1`  
- [ ] Column horizontal align (left/center/right)  
- [ ] Shared-row layout fast path for HFT (optional perf)  
- [ ] Internal links / bookmarks (optional, phase 7)  

---

## Checklist — JSON → model

- [x] Load retail / active / hft JSON  
- [x] Expand active 40 / HFT 2000 with seed  
- [ ] Optional: load financials/summary fully from JSON without recompute  
- [ ] Optional: digital_signature block (engine has no sign yet — out of scope)  

---

## Checklist — cached vs non-cached

| Mode | Env | What is reused | What still runs every iter |
|------|-----|----------------|----------------------------|
| **Cached** | `BENCH_CACHE=1` (default) | `ContractNote` + trade rows | layout + `GenerateDocument` |
| **Uncached** | `BENCH_CACHE=0` | JSON file only | `ExpandTrades` + layout + PDF |

- [x] Implement both  
- [ ] Publish baseline ops/sec for cached vs uncached (compliant + nocomply)  

---

## Makefile targets

```bash
make bench-zerodha                 # compliant, cache ON
make bench-zerodha-nocomply        # PDF 2.0 only
make bench-zerodha-cached          # BENCH_CACHE=1
make bench-zerodha-uncached        # BENCH_CACHE=0
make bench-zerodha-x2 | x5 | x10
make bench-zerodha-nocomply-x10
```

---

## Acceptance

- [ ] `BENCH_ITERATIONS=20 BENCH_WORKERS=4 make bench-zerodha` succeeds  
- [ ] `BENCH_CACHE=0 BENCH_ITERATIONS=20 make bench-zerodha` succeeds  
- [ ] `make bench-zerodha-nocomply` succeeds  
- [ ] Warm-up PDFs written under `sampledata/zerodha/`  
- [ ] HFT multi-page (>1 page) when 2000 trades  
- [ ] Colored header/section/action cells visible in a viewer  

### Later (compliance quality)

- [ ] Compliant warm-up PDFs pass veraPDF `-f 4` and `-f ua2`  
- [ ] structure_tree_check on multipage tables (real TD/TH MCIDs — needs richer UA tagging)  

---

## Explicit non-goals

- gopdfsuit as a generator backend  
- ECDSA/RSA signing in this harness  
- Modifying the gopdfsuit repo  
