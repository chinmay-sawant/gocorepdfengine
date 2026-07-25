// Package color provides PDF color primitives including RGB representation
// and hex color parsing.
package color

import (
	"fmt"
	"strconv"
	"strings"
)

// RGB is a 0–1 RGB triple for PDF content operators (rg / RG).
type RGB [3]float64

// ParseHex parses "#RGB", "#RRGGBB", or "RRGGBB" into an RGB triple in 0–1 range.
func ParseHex(s string) (RGB, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	switch len(s) {
	case 3:
		r, err1 := strconv.ParseUint(string(s[0])+string(s[0]), 16, 8)
		g, err2 := strconv.ParseUint(string(s[1])+string(s[1]), 16, 8)
		b, err3 := strconv.ParseUint(string(s[2])+string(s[2]), 16, 8)
		if err1 != nil || err2 != nil || err3 != nil {
			return RGB{}, fmt.Errorf("invalid hex color %q", s)
		}
		return RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}, nil
	case 6:
		n, err := strconv.ParseUint(s, 16, 32)
		if err != nil {
			return RGB{}, fmt.Errorf("invalid hex color %q: %w", s, err)
		}
		return RGB{
			float64((n>>16)&0xff) / 255,
			float64((n>>8)&0xff) / 255,
			float64(n&0xff) / 255,
		}, nil
	default:
		return RGB{}, fmt.Errorf("invalid hex color length %q", s)
	}
}

// MustHex parses a hex color string and returns the RGB triple.
// It returns an error if s is not a valid hex color.
func MustHex(s string) (RGB, error) {
	return ParseHex(s)
}

// Theme colours for contract-note layout. These are intentional package-level
// configuration constants (BP-37), not mutable global state.
var (
	ThemeHeaderBG   RGB
	ThemeHeaderFG   RGB
	ThemeHeaderSub  RGB
	ThemeSectionBG  RGB
	ThemeSectionFG  RGB
	ThemeTableHead  RGB
	ThemeAltRow     RGB
	ThemeInfoRow    RGB
	ThemeSummaryBG  RGB
	ThemeBuy        RGB
	ThemeSell       RGB
	ThemeLink       RGB
	ThemeBlack      = RGB{0, 0, 0}
	ThemeWhite      = RGB{1, 1, 1}
)

func init() {
	ThemeHeaderBG, _ = MustHex("#154360")
	ThemeHeaderFG, _ = MustHex("#FFFFFF")
	ThemeHeaderSub, _ = MustHex("#AED6F1")
	ThemeSectionBG, _ = MustHex("#21618C")
	ThemeSectionFG, _ = MustHex("#FFFFFF")
	ThemeTableHead, _ = MustHex("#D4E6F1")
	ThemeAltRow, _ = MustHex("#F8F9F9")
	ThemeInfoRow, _ = MustHex("#EBF5FB")
	ThemeSummaryBG, _ = MustHex("#A9CCE3")
	ThemeBuy, _ = MustHex("#27AE60")
	ThemeSell, _ = MustHex("#E74C3C")
	ThemeLink, _ = MustHex("#2E86C1")
}
