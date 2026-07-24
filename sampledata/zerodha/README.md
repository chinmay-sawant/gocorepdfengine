# Zerodha-style benchmark (gocorepdfengine only)

No gopdfsuit dependency. Pipeline:

**Full template field guide (target contract):** [guides/TEMPLATE_REFERENCE.md](../../guides/TEMPLATE_REFERENCE.md)  
**Plan:** [plans/phase-08-zerodha-benchmark.md](../../plans/phase-08-zerodha-benchmark.md)

These JSON files are **domain** fixtures (`client` / `trades`), not the full `config`/`elements` tree. Render maps them into layout tables that match the colors/structure described in the template reference.

```
JSON templates  →  model.ContractNote  →  layout tables (theme colors)  →  engine.GenerateDocument
```

## Templates

| File | Tier | Trades |
|------|------|--------|
| `retail_investor.json` | 80% | from JSON (2) |
| `active_trader.json` | 15% | expanded to **40** at runtime |
| `hft_algo.json` | 5% | expanded to **2000** at runtime |

## Cache modes

| Env | Behavior |
|-----|----------|
| `BENCH_CACHE=1` (default) | Expand trades **once**; reuse models every iteration |
| `BENCH_CACHE=0` | Re-expand trades + rebuild model **every** iteration |

Both modes still **generate a PDF every iteration** (layout + write). Cache only skips model rebuild.

## Compliance modes

| Build | Flags |
|-------|--------|
| default (`!nocomply`) | PDF/A-4 + PDF/UA-2 modes |
| `-tags nocomply` | PDF 2.0 only |

## Make targets

```bash
make bench-zerodha                 # compliant, cache ON
make bench-zerodha-nocomply
BENCH_CACHE=0 make bench-zerodha   # non-cached model path
make bench-zerodha-x2
make bench-zerodha-x5
make bench-zerodha-x10
make bench-zerodha-nocomply-x10
```

## Env

| Variable | Default | Meaning |
|----------|---------|---------|
| `BENCH_ITERATIONS` | 5000 | jobs |
| `BENCH_WORKERS` | 48 | workers |
| `BENCH_CACHE` | 1 | model cache |
| `BENCH_SKIP_WRITE` | — | `1` = skip warm-up PDF files |
| `BENCH_WARMUP` | on | `0` = skip warm-up |
| `BENCH_SEED` | 42 | expand + schedule seed |

## Package map

| Package | Role |
|---------|------|
| `engine/model` | JSON → `ContractNote`, `ExpandTrades` |
| `engine/color` | hex parse + Zerodha theme colors |
| `engine/layout` | tables, fills, borders, multipage flow |
| `engine/render` | note → layout → `GenerateDocument` |
| `engine.GenerateDocument` | multi-page PDF assembly |
