package color

import "testing"

func TestSRGBProfileDictLengthMatchesData(t *testing.T) {
	// Reset is not possible for sync.Once; just call in correct order and reverse
	// to ensure dict Length always matches the returned profile bytes.
	data := SRGBProfile()
	dict := SRGBProfileDict()
	length, ok := dict["/Length"].(int)
	if !ok {
		t.Fatalf("/Length type %T, want int", dict["/Length"])
	}
	if length != len(data) {
		t.Fatalf("sRGB /Length=%d != data len %d", length, len(data))
	}
	if length == 0 {
		t.Fatal("sRGB profile data is empty")
	}

	gdata := GrayProfile()
	gdict := GrayProfileDict()
	glength, ok := gdict["/Length"].(int)
	if !ok {
		t.Fatalf("gray /Length type %T, want int", gdict["/Length"])
	}
	if glength != len(gdata) {
		t.Fatalf("gray /Length=%d != data len %d", glength, len(gdata))
	}
	if glength == 0 {
		t.Fatal("gray profile data is empty")
	}
}

// Dict must self-initialize profile data even when called before Profile().
func TestProfileDictLengthWithoutPriorProfileCall(t *testing.T) {
	// GrayProfileDict calls GrayProfile internally; Length must be non-zero.
	dict := GrayProfileDict()
	length, _ := dict["/Length"].(int)
	if length == 0 {
		t.Fatal("GrayProfileDict /Length is 0 — stream would fail PDF/A length check")
	}
	dict2 := SRGBProfileDict()
	length2, _ := dict2["/Length"].(int)
	if length2 == 0 {
		t.Fatal("SRGBProfileDict /Length is 0 — stream would fail PDF/A length check")
	}
}
