// codehound-ignore-file: BP-27

// Package pdfa provides PDF/A-4 output-intent and color-space dictionary
// builders for embedding ICC profiles and marking the document as compliant
// with ISO 19005-4 (PDF/A-4).
package pdfa

import (
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/doc"
	"github.com/chinmay/gocorepdfengine/engine/write"
)

// OutputIntentDict returns a PDF OutputIntent dictionary referencing an ICC
// profile for PDF/A-4 conformance.
func OutputIntentDict(iccProfileRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/Type":                      "/OutputIntent",
		"/S":                         "/GTS_PDFA1",
		"/OutputConditionIdentifier": write.PDFString("sRGB IEC61966-2.1"),
		"/RegistryName":              write.PDFString("http://www.color.org"),
		"/Info":                      write.PDFString("sRGB IEC61966-2.1"),
		"/DestOutputProfile":         strconv.Itoa(int(iccProfileRef)) + " 0 R",
	}
}

// CatalogA4Extras returns catalog entries needed for PDF/A-4 compliance:
// the OutputIntents array.
func CatalogA4Extras(outputIntentRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/OutputIntents": []interface{}{strconv.Itoa(int(outputIntentRef)) + " 0 R"},
	}
}

// PageResourceA4Extras returns page-level ColorSpace resources mapping
// /DefaultRGB and /DefaultGray to ICC-based profiles.
func PageResourceA4Extras(srgbRef, grayRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/ColorSpace": map[string]interface{}{
			"/DefaultRGB":  []interface{}{"/ICCBased", strconv.Itoa(int(srgbRef)) + " 0 R"},
			"/DefaultGray": []interface{}{"/ICCBased", strconv.Itoa(int(grayRef)) + " 0 R"},
		},
	}
}

// ImageColorSpace returns a PDF color-space reference string suitable for
// use with an ICCBased sRGB profile.
func ImageColorSpace(srgbRef doc.ObjectID) string {
	return "[/ICCBased " + strconv.Itoa(int(srgbRef)) + " 0 R]"
}
