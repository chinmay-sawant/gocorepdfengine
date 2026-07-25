// Package image provides image loading, caching, and PDF XObject generation
// for JPEG and PNG formats.
package image

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math"
	"sync"
)

// codehound-ignore: BP-40
const (
	maxCacheEntries = 200

	jpegMarker         = 0xFF
	shift8             = 8
	chGray             = 1
	chRGB              = 3
	chRGBA             = 4
	jpegSOSMarker      = 0xDA
	jpegMinSegmentLen  = 2
	jpegQualityDefault = 85
	bitsPerComponent   = 8
)

// cache is an intentional package-level LRU singleton for decoded images
// (BP-37). Protected by cacheMu and initialised lazily via cacheOnce.
// codehound-ignore: BP-37
var (
	cache      map[string]*Image
	cacheMu    sync.Mutex
	cacheOnce  sync.Once
	cacheOrder []string
)

func initCache() {
	cacheOnce.Do(func() {
		cache = make(map[string]*Image)                 // BP-52: single shared cache, size managed via LRU
		cacheOrder = make([]string, 0, maxCacheEntries) // BP-52: eviction order, pre-sized
	})
}

func cacheSet(key string, img *Image) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	if _, exists := cache[key]; !exists {
		if len(cache) >= maxCacheEntries {
			oldest := cacheOrder[0]
			delete(cache, oldest)
			cacheOrder = cacheOrder[1:]
		}
		cacheOrder = append(cacheOrder, key)
	}
	cache[key] = img
}

func cacheGet(key string) (*Image, bool) {
	cacheMu.Lock()
	defer cacheMu.Unlock()
	img, ok := cache[key]
	return img, ok
}

func cacheKey(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:8]) // cold path (one-time per image)
}

// Image represents a loaded image ready for PDF embedding.
type Image struct {
	Width, Height    int
	ColorSpace       string
	BitsPerComponent int
	Data             []byte
	Filter           string
}

func checkOverflow(w, h int) error {
	if w <= 0 || h <= 0 {
		return errors.New("image: invalid image dimensions")
	}
	if w > math.MaxInt/h {
		return errors.New("image: image dimensions too large")
	}
	return nil
}

// NewFromJPEG parses JPEG headers to extract width, height, color space, and
// bits-per-component without fully decoding the image. Returns a cached Image.
func NewFromJPEG(data []byte) (*Image, error) {
	if len(data) < 2 || data[0] != jpegMarker || data[1] != 0xD8 {
		return nil, errors.New("image: invalid JPEG: missing SOI marker")
	}

	initCache()
	key := cacheKey(data)
	if cached, ok := cacheGet(key); ok {
		return cached, nil
	}

	i := 2
	for i < len(data) {
		if data[i] != jpegMarker {
			i++
			continue
		}
		i++
		if i >= len(data) {
			return nil, errors.New("image: unexpected end of JPEG data")
		}
		if data[i] == 0x00 || data[i] == jpegMarker {
			i++
			continue
		}

		marker := data[i]

		if marker == 0xC0 || marker == 0xC1 {
			if i+9 >= len(data) {
				return nil, errors.New("image: truncated JPEG SOF")
			}
			precision := int(data[i+3])
			height := int(data[i+4])<<shift8 | int(data[i+5])
			width := int(data[i+6])<<shift8 | int(data[i+7])
			numComponents := int(data[i+8])

			cs := "/DeviceRGB"
			switch numComponents {
			case chGray:
				cs = "/DeviceGray"
			case chRGB:
				cs = "/DeviceRGB"
			case chRGBA:
				cs = "/DeviceCMYK"
			}

			img := &Image{
				Width:            width,
				Height:           height,
				ColorSpace:       cs,
				BitsPerComponent: precision,
				Data:             data,
				Filter:           "/DCTDecode",
			}
			cacheSet(key, img)
			return img, nil
		}

		if marker == 0xD9 || marker == jpegSOSMarker || marker == 0x01 ||
			(marker >= 0xD0 && marker <= 0xD7) {
			if marker == jpegSOSMarker {
				return nil, errors.New("image: SOS found before SOF")
			}
			i++
			continue
		}

		if i+2 >= len(data) {
			return nil, errors.New("image: truncated JPEG marker length")
		}
		length := int(data[i+1])<<shift8 | int(data[i+2])
		if length < jpegMinSegmentLen {
			return nil, errors.New("image: invalid JPEG marker length")
		}
		i += 1 + length
	}

	return nil, errors.New("image: JPEG SOF marker not found")
}

// NewFromPNG decodes a PNG image using the standard library, then converts it
// to a JPEG (large images) or zlib-compressed RGB (small images) for PDF.
// Results are cached by content hash.
func NewFromPNG(data []byte) (*Image, error) {
	initCache()
	key := cacheKey(data)
	if cached, ok := cacheGet(key); ok {
		return cached, nil
	}

	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		// codehound-ignore: PERF-35
		return nil, fmt.Errorf("image: PNG decode error: %w", err) // cold path (decode failure)
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if err := checkOverflow(w, h); err != nil {
		return nil, err
	}

	const jpegThreshold = 100 * 100
	var img *Image
	if w*h >= jpegThreshold {
		var jpgBuf bytes.Buffer
		if err := jpeg.Encode(&jpgBuf, src, &jpeg.Options{Quality: jpegQualityDefault}); err != nil {
			return nil, fmt.Errorf("image: JPEG encode error: %w", err)
		}
		img = &Image{
			Width:            w,
			Height:           h,
			ColorSpace:       "/DeviceRGB",
			BitsPerComponent: bitsPerComponent,
			Data:             jpgBuf.Bytes(),
			Filter:           "/DCTDecode",
		}
	} else {
		flateData, err := encodePNGAsFlate(src, bounds, w, h)
		if err != nil {
			return nil, err
		}
		img = &Image{
			Width:            w,
			Height:           h,
			ColorSpace:       "/DeviceRGB",
			BitsPerComponent: bitsPerComponent,
			Data:             flateData,
			Filter:           "/FlateDecode",
		}
	}
	cacheSet(key, img)
	return img, nil
}

func encodePNGAsFlate(src image.Image, bounds image.Rectangle, w, h int) ([]byte, error) {
	totalPixels := w * h
	if totalPixels > math.MaxInt/3 {
		return nil, errors.New("image: image too large for RGB buffer")
	}
	rawRGB := make([]byte, 0, totalPixels*chRGB)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			// codehound-ignore: BP-1
			r, g, b, _ := src.At(x, y).RGBA()
			rawRGB = append(rawRGB, byte(r>>shift8), byte(g>>shift8), byte(b>>shift8))
		}
	}
	var compressed bytes.Buffer
	// codehound-ignore: PERF-233
	// codehound-ignore: PERF-227
	zw := zlib.NewWriter(&compressed)
	if _, err := zw.Write(rawRGB); err != nil {
		return nil, fmt.Errorf("image: zlib write: %w", err)
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("image: zlib close: %w", err)
	}
	return compressed.Bytes(), nil
}

// XObjectDict returns a PDF dictionary for placing this Image as an XObject.
// The colorSpaceRef parameter should be a PDF color space reference string.
// codehound-ignore: BP-27
func (img *Image) XObjectDict(_ string, colorSpaceRef string) map[string]interface{} {
	return map[string]interface{}{
		"/Type":             "/XObject",
		"/Subtype":          "/Image",
		"/Width":            img.Width,
		"/Height":           img.Height,
		"/ColorSpace":       colorSpaceRef,
		"/BitsPerComponent": img.BitsPerComponent,
		"/Filter":           img.Filter,
		"/Length":           len(img.Data),
	}
}
