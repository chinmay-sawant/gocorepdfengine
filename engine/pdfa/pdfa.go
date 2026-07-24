package pdfa

import (
	"fmt"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

func OutputIntentDict(iccProfileRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/Type":                      "/OutputIntent",
		"/S":                         "/GTS_PDFA1",
		"/OutputConditionIdentifier": "sRGB IEC61966-2.1",
		"/RegistryName":              "http://www.color.org",
		"/Info":                      "sRGB IEC61966-2.1",
		"/DestOutputProfile":         fmt.Sprintf("%d 0 R", iccProfileRef),
	}
}

func CatalogA4Extras(outputIntentRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/OutputIntents": []interface{}{fmt.Sprintf("%d 0 R", outputIntentRef)},
	}
}

func PageResourceA4Extras(srgbRef, grayRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/ColorSpace": map[string]interface{}{
			"/DefaultRGB":  []interface{}{"/ICCBased", fmt.Sprintf("%d 0 R", srgbRef)},
			"/DefaultGray": []interface{}{"/ICCBased", fmt.Sprintf("%d 0 R", grayRef)},
		},
	}
}

func ImageColorSpace(srgbRef doc.ObjectID) string {
	return fmt.Sprintf("[/ICCBased %d 0 R]", srgbRef)
}
