#!/usr/bin/env bash
# Zerodha gold standard: timing ×5 + CPU pprof ×5 + one heap profile (compliant).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT="${BENCH_OUT_DIR:-$REPO_ROOT/baselines/zerodha_pprof_runs}"
ZERODHA="$REPO_ROOT/sampledata/zerodha"
BIN="$OUT/zerodha_bench"

mkdir -p "$OUT"
export BENCH_ITERATIONS="${BENCH_ITERATIONS:-5000}"
export BENCH_WORKERS="${BENCH_WORKERS:-48}"

echo "Zerodha benchmark x5 + pprof: iterations=$BENCH_ITERATIONS workers=$BENCH_WORKERS"
echo "Output: $OUT"

cd "$ZERODHA"
go build -o "$BIN" .

for i in 1 2 3 4 5; do
  echo "=== Run $i / 5 (timing) ==="
  "$BIN" 2>&1 | tee "$OUT/zerodha_run${i}.txt"
done

for i in 1 2 3 4 5; do
  echo "=== CPU profile run $i / 5 ==="
  "$BIN" -cpuprofile="$OUT/cpu_zerodha_run${i}.prof" 2>&1 | tee "$OUT/zerodha_cpu_run${i}.txt"
done

echo "=== Heap profile run ==="
"$BIN" -memprofile="$OUT/heap_zerodha.prof" 2>&1 | tee "$OUT/zerodha_heap.txt"

echo "Done."
