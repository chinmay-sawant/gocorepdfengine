// Zerodha-style gold-standard harness for gocorepdfengine.
//
// Pipeline: JSON templates → model.ContractNote → layout tables (colors) → PDF.
// Modes:
//   - compliant (!nocomply): PDF/A-4 + PDF/UA-2 flags
//   - non-compliant (nocomply): PDF 2.0 only
//
// Cache:
//   - BENCH_CACHE=1 (default): expand trades once; reuse models across iterations
//   - BENCH_CACHE=0: re-expand trades + rebuild model every iteration

// codehound-ignore-file: BP-48,BP-41,CWE-497,PERF-148,PERF-171
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chinmay/gocorepdfengine/engine/model"
	"github.com/chinmay/gocorepdfengine/engine/render"
)

// Monitoring limits
const (
	memMonitorIntervalMs = 100
	bytesPerKB           = 1024
)

// Benchmark parameters
const (
	defaultIterations = 5000
	defaultWorkers    = 48
	defaultBenchSeed  = 42
)

// Trade counts
const (
	activeTraderCount = 40
	hftTraderCount    = 2000
)

// Math constants
// codehound-ignore: BP-40
const (
	usPerMs       = 1000.0
	retailPercent = 80
	activePercent = 15
	percentBase   = 100
)

// RNG constants
const (
	xorShiftA = 13
	xorShiftB = 7
	xorShiftC = 17
)

// Seed offsets
const (
	benchActiveSeedOffset = 1
	benchHFTSeedOffset    = 2
)

// File system
const (
	filePerm = 0o600
)

var (
	flagCPUProfile = flag.String("cpuprofile", "", "write CPU profile to file")
	flagMemProfile = flag.String("memprofile", "", "write heap profile to file")
)

type latencyStats struct {
	count, sumNs, minNs, maxNs int64
}

// set by main.go / main_nocomply.go
var benchCompliant bool

func runMain() {
	flag.Parse()
	var cpuProfileFile *os.File
	if *flagCPUProfile != "" {
		var err error
		cpuProfileFile, err = os.Create(*flagCPUProfile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1) // Benchmark harness, not library code.
		}
		if err := pprof.StartCPUProfile(cpuProfileFile); err != nil {
			// codehound-ignore: BP-5
			// codehound-ignore: BP-1
			_ = cpuProfileFile.Close() // Best-effort cleanup; original error is surfaced below.
			fmt.Println(err)
			os.Exit(1) // Benchmark harness, not library code.
		}
	}
	if err := runBenchmark(); err != nil {
		fmt.Println(err)
		if cpuProfileFile != nil {
			pprof.StopCPUProfile()
			// codehound-ignore: BP-5
			// codehound-ignore: BP-1
			_ = cpuProfileFile.Close() // Best-effort; original error is surfaced above.
		}
		os.Exit(1) // Benchmark harness, not library code.
	}
	if cpuProfileFile != nil {
		pprof.StopCPUProfile()
		// codehound-ignore: BP-5
		// codehound-ignore: BP-1
		_ = cpuProfileFile.Close() // Best-effort cleanup.
	}
	if *flagMemProfile != "" {
		f, err := os.Create(*flagMemProfile)
		if err != nil {
			fmt.Println(err)
			os.Exit(1) // Benchmark harness, not library code.
		}
		// codehound-ignore: BP-1
		_ = pprof.WriteHeapProfile(f) // Diagnostic — error discarded intentionally.
		// codehound-ignore: BP-1
		// codehound-ignore: BP-5
		_ = f.Close() // Best-effort cleanup after heap profile write.
	}
}

func envInt(key string, fallback int) int {
	if raw := os.Getenv(key); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

func envCacheEnabled() bool {
	// Default ON: cached models (template expand once).
	v := os.Getenv("BENCH_CACHE")
	if v == "" {
		return true
	}
	return v != "0" && v != "false" && v != "off"
}

func loadBaseNotes() (*model.ContractNote, *model.ContractNote, *model.ContractNote, error) {
	// Templates live next to this package.
	dir, err := os.Getwd()
	if err != nil {
		return nil, nil, nil, errf("getwd", err)
	}
	retail, err := model.LoadJSON(filepath.Join(dir, "retail_investor.json"))
	if err != nil {
		return nil, nil, nil, errf("load retail", err)
	}
	active, err := model.LoadJSON(filepath.Join(dir, "active_trader.json"))
	if err != nil {
		return nil, nil, nil, errf("load active", err)
	}
	hft, err := model.LoadJSON(filepath.Join(dir, "hft_algo.json"))
	if err != nil {
		return nil, nil, nil, errf("load hft", err)
	}
	return retail, active, hft, nil
}

func prepareNote(base *model.ContractNote, tradeCount int, seed int64) *model.ContractNote {
	n := base.Clone()
	if tradeCount > 0 {
		n.ExpandTrades(tradeCount, seed)
	}
	return n
}

func renderNote(n *model.ContractNote) ([]byte, error) {
	pdf, err := render.PDF(n, render.Options{Compliant: benchCompliant})
	if err != nil {
		return nil, errf("render note", err)
	}
	return pdf, nil
}

func monitorMemory(done chan bool, wg *sync.WaitGroup) {
	defer wg.Done()
	var maxAlloc uint64
	ticker := time.NewTicker(memMonitorIntervalMs * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			fmt.Printf("  Max Memory Allocated: %.2f MB\n", float64(maxAlloc)/bytesPerKB/bytesPerKB)
			return
		case <-ticker.C:
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			if m.Alloc > maxAlloc {
				maxAlloc = m.Alloc
			}
		}
	}
}

func runBenchmark() error {
	fmt.Println("=== Zerodha Gold Standard Benchmark (gocorepdfengine) ===")
	fmt.Println("Pipeline: JSON → model → layout (colors) → GenerateDocument")
	if benchCompliant {
		fmt.Println("Mode: compliant (PDF/A-4 + PDF/UA-2 flags)")
	} else {
		fmt.Println("Mode: non-compliant (PDF 2.0 only)")
	}
	cached := envCacheEnabled()
	if cached {
		fmt.Println("Cache: ON  (models/trades expanded once; reused per iteration)")
	} else {
		fmt.Println("Cache: OFF (re-expand trades + rebuild model every iteration)")
	}
	fmt.Println("Workload Mix: 80% Retail | 15% Active | 5% HFT")
	fmt.Println()

	iterations := envInt("BENCH_ITERATIONS", defaultIterations)
	numWorkers := envInt("BENCH_WORKERS", defaultWorkers)
	skipWrite := os.Getenv("BENCH_SKIP_WRITE") == "1"
	benchSeed := int64(defaultBenchSeed) // Fixed seed for deterministic benchmark reproducibility (not security-sensitive).
	if raw := os.Getenv("BENCH_SEED"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
			benchSeed = n
		}
	}

	// Diagnostic output for benchmarking context — exposes OS/arch/Go version intentionally.
	// codehound-ignore: CWE-497
	fmt.Printf("OS: %s, Arch: %s, NumCPU: %d, Go: %s\n",
		runtime.GOOS, runtime.GOARCH, runtime.NumCPU(), runtime.Version())
	fmt.Printf("GOMAXPROCS: %d\n", runtime.GOMAXPROCS(0))
	fmt.Printf("Running %d iterations using %d workers...\n\n", iterations, numWorkers)

	baseRetail, baseActive, baseHFT, err := loadBaseNotes()
	if err != nil {
		return errf("loading base notes", err)
	}

	// Cached models (expanded once).
	var retailNote, activeNote, hftNote *model.ContractNote
	if cached {
		fmt.Println("Building cached models (JSON + ExpandTrades)...")
		retailNote = prepareNote(baseRetail, 0, benchSeed)
		activeNote = prepareNote(baseActive, activeTraderCount, benchSeed+benchActiveSeedOffset)
		hftNote = prepareNote(baseHFT, hftTraderCount, benchSeed+benchHFTSeedOffset)
		fmt.Printf("  Retail trades: %d | Active: %d | HFT: %d\n",
			len(retailNote.Trades), len(activeNote.Trades), len(hftNote.Trades))
	}

	opts := render.Options{Compliant: benchCompliant}
	var retailPDF, activePDF, hftPDF []byte
	if err := runWarmup(cached, baseRetail, baseActive, baseHFT, benchSeed, opts, retailNote, activeNote, hftNote, &retailPDF, &activePDF, &hftPDF); err != nil {
		return errf("warmup", err)
	}

	const (
		workloadRetail = iota
		workloadActive
		workloadHFT
	)
	schedule := buildSchedule(iterations, benchSeed)

	jobs := make(chan int, iterations)
	errCh := make(chan error, iterations)
	workerStats := make([]latencyStats, numWorkers)
	var retailCount, activeCount, hftCount int64

	memDone := make(chan bool, 1)
	var memWg sync.WaitGroup
	memWg.Add(1)
	go monitorMemory(memDone, &memWg)

	var wg sync.WaitGroup
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			stats := &workerStats[workerID]
			for jobIdx := range jobs {
				var note *model.ContractNote
				switch schedule[jobIdx] {
				case workloadRetail:
					atomic.AddInt64(&retailCount, 1)
					if cached {
						note = retailNote
					} else {
						note = prepareNote(baseRetail, 0, benchSeed+int64(jobIdx))
					}
				case workloadActive:
					atomic.AddInt64(&activeCount, 1)
					if cached {
						note = activeNote
					} else {
						note = prepareNote(baseActive, activeTraderCount, benchSeed+int64(jobIdx))
					}
				default:
					atomic.AddInt64(&hftCount, 1)
					if cached {
						note = hftNote
					} else {
						note = prepareNote(baseHFT, hftTraderCount, benchSeed+int64(jobIdx))
					}
				}

				start := time.Now()
				_, err := renderNote(note)
				elapsed := time.Since(start)
				if err != nil {
					errCh <- err
					continue
				}
				ns := elapsed.Nanoseconds()
				stats.count++
				stats.sumNs += ns
				if stats.count == 1 || ns < stats.minNs {
					stats.minNs = ns
				}
				if ns > stats.maxNs {
					stats.maxNs = ns
				}
			}
			wg.Done()
		}(w)
	}

	totalStart := time.Now()
	for i := 0; i < iterations; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	totalTime := time.Since(totalStart)
	memDone <- true
	memWg.Wait()
	close(errCh)

	var errCount int
	for e := range errCh {
		if errCount == 0 {
			fmt.Printf("  First error: %v\n", e)
		}
		errCount++
	}
	if errCount > 0 {
		return errfs("benchmark failed with %d errors", errCount)
	}

	var totalCount, totalSumNs, minNs, maxNs int64
	totalCount, totalSumNs, minNs, maxNs = aggregateStats(workerStats)
	if totalCount == 0 {
		return errors.New("no results collected")
	}

	avgDuration := time.Duration(totalSumNs / totalCount)
	opsPerSec := float64(iterations) / totalTime.Seconds()

	fmt.Println("=== Performance Summary ===")
	fmt.Printf("  Iterations:      %d\n", iterations)
	fmt.Printf("  Concurrency:     %d workers\n", numWorkers)
	fmt.Printf("  Cache:           %v\n", cached)
	fmt.Printf("  Compliant:       %v\n", benchCompliant)
	fmt.Printf("  Total time:      %.3f s\n", totalTime.Seconds())
	fmt.Printf("  Throughput:      %.2f ops/sec\n", opsPerSec)
	fmt.Println()
	fmt.Printf("  Avg Latency:     %.3f ms\n", float64(avgDuration.Microseconds())/usPerMs)
	fmt.Printf("  Min Latency:     %.3f ms\n", float64(time.Duration(minNs).Microseconds())/usPerMs)
	fmt.Printf("  Max Latency:     %.3f ms\n", float64(time.Duration(maxNs).Microseconds())/usPerMs)
	fmt.Println()
	fmt.Println("=== Workload Distribution ===")
	fmt.Printf("  Retail  (80%%):   %d\n", atomic.LoadInt64(&retailCount))
	fmt.Printf("  Active  (15%%):   %d\n", atomic.LoadInt64(&activeCount))
	fmt.Printf("  HFT      (5%%):   %d\n", atomic.LoadInt64(&hftCount))
	fmt.Println()

	if !skipWrite && retailPDF != nil {
		for name, data := range map[string][]byte{
			outputName("zerodha_retail_output.pdf"): retailPDF,
			outputName("zerodha_active_output.pdf"): activePDF,
			outputName("zerodha_hft_output.pdf"):    hftPDF,
		} {
			if err := os.WriteFile(name, data, filePerm); err != nil {
				fmt.Printf("Error saving %s: %v\n", name, err)
			} else {
				fmt.Printf("Saved: %s (%d bytes)\n", name, len(data))
			}
		}
	}

	fmt.Println("=== Done ===")
	return nil
}

func buildSchedule(iterations int, benchSeed int64) []int {
	schedule := make([]int, iterations)
	retailTarget := iterations * retailPercent / percentBase
	activeTarget := iterations * activePercent / percentBase
	for i := range schedule {
		switch {
		case i < retailTarget:
			schedule[i] = 0
		case i < retailTarget+activeTarget:
			schedule[i] = 1
		default:
			schedule[i] = 2
		}
	}
	rng := new(simpleRNG)
	rng.seed = uint64(benchSeed)
	for i := len(schedule) - 1; i > 0; i-- {
		j := int(rng.next() % uint64(i+1))
		schedule[i], schedule[j] = schedule[j], schedule[i]
	}
	return schedule
}

func runWarmup(cached bool, baseRetail, baseActive, baseHFT *model.ContractNote, benchSeed int64, opts render.Options, retailNote, activeNote, hftNote *model.ContractNote, retailPDF, activePDF, hftPDF *[]byte) error {
	if os.Getenv("BENCH_WARMUP") == "0" {
		return nil
	}
	fmt.Println("Warm-up runs...")
	r := retailNote
	a := activeNote
	h := hftNote
	if !cached {
		r = prepareNote(baseRetail, 0, benchSeed)
		a = prepareNote(baseActive, activeTraderCount, benchSeed+benchActiveSeedOffset)
		h = prepareNote(baseHFT, hftTraderCount, benchSeed+benchHFTSeedOffset)
	}
	var err error
	*retailPDF, err = render.PDF(r, opts)
	if err != nil {
		return errf("retail warm-up", err)
	}
	*activePDF, err = render.PDF(a, opts)
	if err != nil {
		return errf("active warm-up", err)
	}
	*hftPDF, err = render.PDF(h, opts)
	if err != nil {
		return errf("hft warm-up", err)
	}
	fmt.Printf("  Retail PDF: %d bytes (%.2f KB)\n", len(*retailPDF), float64(len(*retailPDF))/bytesPerKB)
	fmt.Printf("  Active PDF: %d bytes (%.2f KB)\n", len(*activePDF), float64(len(*activePDF))/bytesPerKB)
	fmt.Printf("  HFT PDF:    %d bytes (%.2f KB)\n", len(*hftPDF), float64(len(*hftPDF))/bytesPerKB)
	fmt.Println()
	return nil
}

func aggregateStats(workerStats []latencyStats) (int64, int64, int64, int64) {
	var totalCount, totalSumNs, minNs, maxNs int64
	for _, stats := range workerStats {
		if stats.count == 0 {
			continue
		}
		totalCount += stats.count
		totalSumNs += stats.sumNs
		if minNs == 0 || (stats.minNs > 0 && stats.minNs < minNs) {
			minNs = stats.minNs
		}
		if stats.maxNs > maxNs {
			maxNs = stats.maxNs
		}
	}
	return totalCount, totalSumNs, minNs, maxNs
}

func outputName(base string) string {
	if benchCompliant {
		return base
	}
	// zerodha_retail_output.pdf → zerodha_retail_nocomply_output.pdf
	if len(base) > 11 && base[len(base)-11:] == "_output.pdf" {
		return base[:len(base)-11] + "_nocomply_output.pdf"
	}
	return "nocomply_" + base
}

// tiny xorshift for schedule shuffle (no math/rand import needed for schedule).
type simpleRNG struct{ seed uint64 }

func (r *simpleRNG) next() uint64 {
	r.seed ^= r.seed << xorShiftA
	r.seed ^= r.seed >> xorShiftB
	r.seed ^= r.seed << xorShiftC
	if r.seed == 0 {
		r.seed = 1
	}
	return r.seed
}

func errf(msg string, err error) error {
	return errors.Join(errors.New(msg), err)
}

func errfs(format string, args ...any) error {
	// codehound-ignore: PERF-35
	return fmt.Errorf(format, args...)
}
