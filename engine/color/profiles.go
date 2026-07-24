package color

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

var (
	srgbOnce sync.Once
	grayOnce sync.Once
	srgbData []byte
	grayData []byte
)

var zlibWriterPool = sync.Pool{
	New: func() any {
		w, err := zlib.NewWriterLevel(io.Discard, flate.BestSpeed)
		if err != nil {
			panic(err)
		}
		return w
	},
}

func loadOrBuildProfile(path string, build func() []byte) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		return compress(data)
	}
	return compress(build())
}

// SRGBProfile returns the compressed sRGB ICC profile data.
func SRGBProfile() []byte {
	srgbOnce.Do(func() {
		var err error
		srgbData, err = loadOrBuildProfile("/usr/share/color/icc/ghostscript/scrgb.icc", buildSRGB)
		if err != nil {
			srgbData = nil
		}
	})
	return srgbData
}

var _ = buildSRGB
var _ = buildGray

// GrayProfile returns the compressed gray ICC profile data.
func GrayProfile() []byte {
	grayOnce.Do(func() {
		var err error
		grayData, err = loadOrBuildProfile("/usr/share/color/icc/ghostscript/sgray.icc", buildGray)
		if err != nil {
			grayData = nil
		}
	})
	return grayData
}

// SRGBProfileDict returns the PDF stream dictionary for the sRGB ICC profile.
func SRGBProfileDict() map[string]interface{} {
	return map[string]interface{}{
		"/N":         3,
		"/Alternate": "/DeviceRGB",
		"/Filter":    "/FlateDecode",
		"/Length":    len(srgbData),
	}
}

// GrayProfileDict returns the PDF stream dictionary for the gray ICC profile.
func GrayProfileDict() map[string]interface{} {
	return map[string]interface{}{
		"/N":         1,
		"/Alternate": "/DeviceGray",
		"/Filter":    "/FlateDecode",
		"/Length":    len(grayData),
	}
}

func s15Fixed16(v float64) int32 {
	return int32(v * 65536.0)
}

func align4(n int) int {
	return (n + 3) & ^3
}

type iccTag struct {
	sig  string
	data []byte
}

func buildHeader(size int, deviceClass, colorSpace, pcs string) []byte {
	hdr := make([]byte, 128)
	binary.BigEndian.PutUint32(hdr[0:4], uint32(size))
	copy(hdr[4:8], "appl")
	binary.BigEndian.PutUint32(hdr[8:12], 0x02100000)
	copy(hdr[12:16], deviceClass)
	copy(hdr[16:20], colorSpace)
	copy(hdr[20:24], pcs)
	dt := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	binary.BigEndian.PutUint16(hdr[24:26], uint16(dt.Year()))
	binary.BigEndian.PutUint16(hdr[26:28], uint16(dt.Month()))
	binary.BigEndian.PutUint16(hdr[28:30], uint16(dt.Day()))
	binary.BigEndian.PutUint16(hdr[30:32], uint16(dt.Hour()))
	binary.BigEndian.PutUint16(hdr[32:34], uint16(dt.Minute()))
	binary.BigEndian.PutUint16(hdr[34:36], uint16(dt.Second()))
	copy(hdr[36:40], "acsp")
	copy(hdr[40:44], "APPL")
	binary.BigEndian.PutUint32(hdr[64:68], 0)
	binary.BigEndian.PutUint32(hdr[68:72], uint32(s15Fixed16(0.9642)))
	binary.BigEndian.PutUint32(hdr[72:76], uint32(s15Fixed16(1.0)))
	binary.BigEndian.PutUint32(hdr[76:80], uint32(s15Fixed16(0.8249)))
	copy(hdr[80:84], "appl")
	return hdr
}

func buildICCProfile(deviceClass, colorSpace, pcs string, tags []iccTag) []byte {
	tagTableSize := 4 + len(tags)*12
	dataStart := 128 + tagTableSize
	totalDataSize := 0
	for _, t := range tags {
		totalDataSize += align4(len(t.data))
	}
	size := dataStart + totalDataSize

	hdr := buildHeader(size, deviceClass, colorSpace, pcs)
	buf := bytes.NewBuffer(hdr)

	buf.Write([]byte{
		byte(len(tags) >> 24), byte(len(tags) >> 16), byte(len(tags) >> 8), byte(len(tags)),
	})
	offset := uint32(dataStart)
	for _, t := range tags {
		buf.Write([]byte{t.sig[0], t.sig[1], t.sig[2], t.sig[3]})
		buf.Write([]byte{
			byte(offset >> 24), byte(offset >> 16), byte(offset >> 8), byte(offset),
		})
		buf.Write([]byte{
			byte(len(t.data) >> 24), byte(len(t.data) >> 16), byte(len(t.data) >> 8), byte(len(t.data)),
		})
		offset += uint32(align4(len(t.data)))
	}
	for _, t := range tags {
		buf.Write(t.data)
		for buf.Len()%4 != 0 {
			buf.WriteByte(0)
		}
	}
	return buf.Bytes()
}

func buildDesc(text string) []byte {
	var buf bytes.Buffer
	buf.Grow(20 + len(text))
	buf.Write([]byte("desc"))
	binary.Write(&buf, binary.BigEndian, uint32(0))   //nolint: errcheck
	asciiCount := uint32(len(text) + 1)
	binary.Write(&buf, binary.BigEndian, asciiCount)  //nolint: errcheck
	buf.WriteString(text)
	buf.WriteByte(0)
	binary.Write(&buf, binary.BigEndian, uint32(0))   //nolint: errcheck
	binary.Write(&buf, binary.BigEndian, uint16(0))  //nolint: errcheck
	return buf.Bytes()
}

func buildXYZ(x, y, z float64) []byte {
	var buf bytes.Buffer
	buf.Write([]byte("XYZ "))
	binary.Write(&buf, binary.BigEndian, uint32(0))         //nolint: errcheck
	binary.Write(&buf, binary.BigEndian, s15Fixed16(x))     //nolint: errcheck
	binary.Write(&buf, binary.BigEndian, s15Fixed16(y))     //nolint: errcheck
	binary.Write(&buf, binary.BigEndian, s15Fixed16(z))     //nolint: errcheck
	return buf.Bytes()
}

func buildCurve(gamma float64) []byte {
	var buf bytes.Buffer
	buf.Write([]byte("curv"))
	binary.Write(&buf, binary.BigEndian, uint32(0))                //nolint: errcheck
	binary.Write(&buf, binary.BigEndian, uint32(1))                //nolint: errcheck
	binary.Write(&buf, binary.BigEndian, uint16(gamma*256.0+0.5))  //nolint: errcheck
	return buf.Bytes()
}

func buildSRGB() []byte {
	return buildICCProfile("mntr", "RGB ", "XYZ ",
		[]iccTag{
			{"desc", buildDesc("sRGB IEC61966-2.1")},
			{"cprt", buildDesc("Copyright (c)")},
			{"wtpt", buildXYZ(0.9642, 1.0, 0.8249)},
			{"rXYZ", buildXYZ(0.4361, 0.2225, 0.0139)},
			{"gXYZ", buildXYZ(0.3851, 0.7169, 0.0971)},
			{"bXYZ", buildXYZ(0.1431, 0.0606, 0.7140)},
			{"rTRC", buildCurve(2.2)},
			{"gTRC", buildCurve(2.2)},
			{"bTRC", buildCurve(2.2)},
		})
}

func buildGray() []byte {
	return buildICCProfile("mntr", "GRAY", "XYZ ",
		[]iccTag{
			{"desc", buildDesc("Gray ICC profile")},
			{"cprt", buildDesc("Copyright (c)")},
			{"wtpt", buildXYZ(0.9642, 1.0, 0.8249)},
			{"kTRC", buildCurve(2.2)},
		})
}

func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	w := zlibWriterPool.Get().(*zlib.Writer)
	defer zlibWriterPool.Put(w)
	w.Reset(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, fmt.Errorf("compress write: %w", err)
	}
	err = w.Close()
	if err != nil {
		return nil, fmt.Errorf("compress close: %w", err)
	}
	return buf.Bytes(), nil
}
