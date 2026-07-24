package image

import (
	"bytes"
	"compress/zlib"
	"crypto/sha256"
	"errors"
	"fmt"
	"image/jpeg"
	"image/png"
	"sync"
)

type Image struct {
	Width, Height      int
	ColorSpace         string
	BitsPerComponent   int
	Data               []byte
	Filter             string
}

var (
	cache   map[string]*Image
	cacheMu sync.Mutex
	cacheOnce sync.Once
)

func initCache() {
	cacheOnce.Do(func() {
		cache = make(map[string]*Image)
	})
}

func cacheKey(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:8])
}

func NewFromJPEG(data []byte) (*Image, error) {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, errors.New("image: invalid JPEG: missing SOI marker")
	}

	initCache()
	key := cacheKey(data)
	cacheMu.Lock()
	if cached, ok := cache[key]; ok {
		cacheMu.Unlock()
		return cached, nil
	}
	cacheMu.Unlock()

	i := 2
	for i < len(data) {
		if data[i] != 0xFF {
			i++
			continue
		}
		i++
		if i >= len(data) {
			return nil, errors.New("image: unexpected end of JPEG data")
		}
		if data[i] == 0x00 || data[i] == 0xFF {
			i++
			continue
		}

		marker := data[i]

		if marker == 0xC0 || marker == 0xC1 {
			if i+9 >= len(data) {
				return nil, errors.New("image: truncated JPEG SOF")
			}
			precision := int(data[i+3])
			height := int(data[i+4])<<8 | int(data[i+5])
			width := int(data[i+6])<<8 | int(data[i+7])
			numComponents := int(data[i+8])

			cs := "/DeviceRGB"
			switch numComponents {
			case 1:
				cs = "/DeviceGray"
			case 3:
				cs = "/DeviceRGB"
			case 4:
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
			cacheMu.Lock()
			cache[key] = img
			cacheMu.Unlock()
			return img, nil
		}

		if marker == 0xD9 || marker == 0xDA || marker == 0x01 ||
			(marker >= 0xD0 && marker <= 0xD7) {
			if marker == 0xDA {
				return nil, errors.New("image: SOS found before SOF")
			}
			i++
			continue
		}

		if i+2 >= len(data) {
			return nil, errors.New("image: truncated JPEG marker length")
		}
		length := int(data[i+1])<<8 | int(data[i+2])
		if length < 2 {
			return nil, errors.New("image: invalid JPEG marker length")
		}
		i += 1 + length
	}

	return nil, errors.New("image: JPEG SOF marker not found")
}

func NewFromPNG(data []byte) (*Image, error) {
	initCache()
	key := cacheKey(data)
	cacheMu.Lock()
	if cached, ok := cache[key]; ok {
		cacheMu.Unlock()
		return cached, nil
	}
	cacheMu.Unlock()

	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("image: PNG decode error: %w", err)
	}

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	// For large images, use JPEG encoding which GS handles reliably.
	const jpegThreshold = 100 * 100 // 100x100 pixels
	var img *Image
	if w*h >= jpegThreshold {
		var jpgBuf bytes.Buffer
		if err := jpeg.Encode(&jpgBuf, src, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("image: JPEG encode error: %w", err)
		}
		img = &Image{
			Width:            w,
			Height:           h,
			ColorSpace:       "/DeviceRGB",
			BitsPerComponent: 8,
			Data:             jpgBuf.Bytes(),
			Filter:           "/DCTDecode",
		}
	} else {
		rawRGB := make([]byte, 0, w*h*3)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r, g, b, _ := src.At(x, y).RGBA()
				rawRGB = append(rawRGB, byte(r>>8), byte(g>>8), byte(b>>8))
			}
		}
		var compressed bytes.Buffer
		zw := zlib.NewWriter(&compressed)
		if _, err := zw.Write(rawRGB); err != nil {
			return nil, err
		}
		if err := zw.Close(); err != nil {
			return nil, err
		}
		img = &Image{
			Width:            w,
			Height:           h,
			ColorSpace:       "/DeviceRGB",
			BitsPerComponent: 8,
			Data:             compressed.Bytes(),
			Filter:           "/FlateDecode",
		}
	}
	cacheMu.Lock()
	cache[key] = img
	cacheMu.Unlock()
	return img, nil
}

func (img *Image) XObjectDict(name, colorSpaceRef string) map[string]interface{} {
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
