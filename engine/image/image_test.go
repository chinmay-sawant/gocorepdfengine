package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func TestNewFromJPEG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}

	parsed, err := NewFromJPEG(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Width != 1 {
		t.Errorf("expected Width=1, got %d", parsed.Width)
	}
	if parsed.Height != 1 {
		t.Errorf("expected Height=1, got %d", parsed.Height)
	}
	if parsed.Filter != "/DCTDecode" {
		t.Errorf("expected Filter=/DCTDecode, got %s", parsed.Filter)
	}
}

func TestNewFromJPEG_Invalid(t *testing.T) {
	_, err := NewFromJPEG([]byte{0x00, 0x00})
	if err == nil {
		t.Error("expected error for invalid JPEG data")
	}
}

func TestNewFromPNG(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{255, 0, 0, 255})
	img.Set(1, 0, color.RGBA{0, 255, 0, 255})
	img.Set(0, 1, color.RGBA{0, 0, 255, 255})
	img.Set(1, 1, color.RGBA{255, 255, 255, 255})

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	parsed, err := NewFromPNG(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Width != 2 {
		t.Errorf("expected Width=2, got %d", parsed.Width)
	}
	if parsed.Height != 2 {
		t.Errorf("expected Height=2, got %d", parsed.Height)
	}
	if parsed.Filter != "/FlateDecode" {
		t.Errorf("expected Filter=/FlateDecode, got %s", parsed.Filter)
	}
	if parsed.BitsPerComponent != 8 {
		t.Errorf("expected BitsPerComponent=8, got %d", parsed.BitsPerComponent)
	}
}

func TestXObjectDict(t *testing.T) {
	img := &Image{
		Width:            100,
		Height:           200,
		ColorSpace:       "/DeviceRGB",
		BitsPerComponent: 8,
		Data:             []byte{0x00, 0x01, 0x02},
		Filter:           "/DCTDecode",
	}

	dict := img.XObjectDict("Im0", "/DeviceRGB")
	expectedKeys := []string{"/Type", "/Subtype", "/Width", "/Height", "/ColorSpace", "/BitsPerComponent", "/Filter", "/Length"}
	for _, k := range expectedKeys {
		if _, ok := dict[k]; !ok {
			t.Errorf("expected key %s not found in dict", k)
		}
	}
	if dict["/Type"] != "/XObject" {
		t.Errorf("expected /Type=/XObject, got %v", dict["/Type"])
	}
	if dict["/Subtype"] != "/Image" {
		t.Errorf("expected /Subtype=/Image, got %v", dict["/Subtype"])
	}
	if dict["/Width"] != 100 {
		t.Errorf("expected /Width=100, got %v", dict["/Width"])
	}
	if dict["/Height"] != 200 {
		t.Errorf("expected /Height=200, got %v", dict["/Height"])
	}
	if dict["/Length"] != 3 {
		t.Errorf("expected /Length=3, got %v", dict["/Length"])
	}
}

func TestDedup(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.White)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}

	first, err := NewFromJPEG(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewFromJPEG(buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("expected same pointer for deduplicated images")
	}
}
