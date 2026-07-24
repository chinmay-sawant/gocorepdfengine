//go:build !nocomply

// Compliant Zerodha-style benchmark (PDF/A-4 + PDF/UA-2 flags).
// Uses gocorepdfengine layout + model JSON templates only (no gopdfsuit).
//
//	make bench-zerodha
//	BENCH_CACHE=0 make bench-zerodha   # rebuild model/trades every iteration
package main

func init() {
	benchCompliant = true
}

func main() {
	runMain()
}
