//go:build nocomply

// Non-compliant Zerodha-style benchmark (PDF 2.0 only, no A-4/UA-2).
// Same JSON templates + layout path as compliant mode.
//
//	make bench-zerodha-nocomply
package main

func init() {
	benchCompliant = false
}

func main() {
	runMain()
}
