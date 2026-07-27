package color

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"os"
	"time"
)

var (
	srgbData []byte
	grayData []byte
)

func init() {
	srgbData = loadOrBuildProfile("/usr/share/color/icc/ghostscript/scrgb.icc", buildSRGB)
	grayData = loadOrBuildProfile("/usr/share/color/icc/ghostscript/sgray.icc", buildGray)
}

func loadOrBuildProfile(path string, build func() []byte) []byte {
	data, err := os.ReadFile(path)
	if err == nil && len(data) > 0 {
		return compress(data)
	}
	return compress(build())
}

func SRGBProfile() []byte {
	return srgbData
}

func GrayProfile() []byte {
	return grayData
}

func SRGBProfileDict() map[string]interface{} {
	return map[string]interface{}{
		"/N":         3,
		"/Alternate": "/DeviceRGB",
		"/Filter":    "/FlateDecode",
		"/Length":    len(srgbData),
	}
}

func GrayProfileDict() map[string]interface{} {
	return map[string]interface{}{
		"/N":         1,
		"/Alternate": "/DeviceGray",
		"/Filter":    "/FlateDecode",
		"/Length":    len(grayData),
	}
}

func s15Fixed16(v float64) int32 {
	return int32(v * 65536.0) //nolint:mnd
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
	binary.BigEndian.PutUint32(hdr[8:12], 0x02100000) //nolint:mnd
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
	binary.BigEndian.PutUint32(hdr[68:72], uint32(s15Fixed16(0.9642))) //nolint:mnd
	binary.BigEndian.PutUint32(hdr[72:76], uint32(s15Fixed16(1.0)))
	binary.BigEndian.PutUint32(hdr[76:80], uint32(s15Fixed16(0.8249))) //nolint:mnd
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

	_ = binary.Write(buf, binary.BigEndian, uint32(len(tags)))
	offset := uint32(dataStart)
	for _, t := range tags {
		_ = binary.Write(buf, binary.BigEndian, [4]byte{t.sig[0], t.sig[1], t.sig[2], t.sig[3]})
		_ = binary.Write(buf, binary.BigEndian, offset)
		_ = binary.Write(buf, binary.BigEndian, uint32(len(t.data)))
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
	buf.Write([]byte("desc"))
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))
	asciiCount := uint32(len(text) + 1)
	_ = binary.Write(&buf, binary.BigEndian, asciiCount)
	buf.WriteString(text)
	buf.WriteByte(0)
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))
	_ = binary.Write(&buf, binary.BigEndian, uint16(0))
	return buf.Bytes()
}

func buildXYZ(x, y, z float64) []byte {
	var buf bytes.Buffer
	buf.Write([]byte("XYZ "))
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))
	_ = binary.Write(&buf, binary.BigEndian, s15Fixed16(x))
	_ = binary.Write(&buf, binary.BigEndian, s15Fixed16(y))
	_ = binary.Write(&buf, binary.BigEndian, s15Fixed16(z))
	return buf.Bytes()
}

func buildCurve() []byte {
	var gamma = 2.2
	var buf bytes.Buffer
	buf.Write([]byte("curv"))
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))
	_ = binary.Write(&buf, binary.BigEndian, uint32(1))
	_ = binary.Write(&buf, binary.BigEndian, uint16(gamma*256.0+0.5))
	return buf.Bytes()
}

func buildSRGB() []byte {
	return buildICCProfile("mntr", "RGB ", "XYZ ",
		[]iccTag{
			{"desc", buildDesc("sRGB IEC61966-2.1")},
			{"cprt", buildDesc("Copyright (c)")},
			{"wtpt", buildXYZ(0.9642, 1.0, 0.8249)},    //nolint:mnd
			{"rXYZ", buildXYZ(0.4361, 0.2225, 0.0139)}, //nolint:mnd
			{"gXYZ", buildXYZ(0.3851, 0.7169, 0.0971)}, //nolint:mnd
			{"bXYZ", buildXYZ(0.1431, 0.0606, 0.7140)}, //nolint:mnd
			{"rTRC", buildCurve()},
			{"gTRC", buildCurve()},
			{"bTRC", buildCurve()},
		})
}

func buildGray() []byte {
	return buildICCProfile("mntr", "GRAY", "XYZ ",
		[]iccTag{
			{"desc", buildDesc("Gray ICC profile")},
			{"cprt", buildDesc("Copyright (c)")},
			{"wtpt", buildXYZ(0.9642, 1.0, 0.8249)}, //nolint:mnd
			{"kTRC", buildCurve()},
		})
}

func compress(data []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	_, _ = w.Write(data)
	w.Close()
	return buf.Bytes()
}
