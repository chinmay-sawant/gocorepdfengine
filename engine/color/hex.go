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

// MustHex panics on parse error (for static theme constants).
func MustHex(s string) RGB {
	c, err := ParseHex(s)
	if err != nil {
		panic(err)
	}
	return c
}

// Zerodha-style theme used by contract-note layout.
var (
	ThemeHeaderBG   = MustHex("#154360")
	ThemeHeaderFG   = MustHex("#FFFFFF")
	ThemeHeaderSub  = MustHex("#AED6F1")
	ThemeSectionBG  = MustHex("#21618C")
	ThemeSectionFG  = MustHex("#FFFFFF")
	ThemeTableHead  = MustHex("#D4E6F1")
	ThemeAltRow     = MustHex("#F8F9F9")
	ThemeInfoRow    = MustHex("#EBF5FB")
	ThemeSummaryBG  = MustHex("#A9CCE3")
	ThemeBuy        = MustHex("#27AE60")
	ThemeSell       = MustHex("#E74C3C")
	ThemeLink       = MustHex("#2E86C1")
	ThemeBlack      = RGB{0, 0, 0}
	ThemeWhite      = RGB{1, 1, 1}
)
