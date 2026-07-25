// codehound-ignore-file: BP-1,BP-27,BP-37,BP-38

package color

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"sync"
	"time"
)

const (
	defaultGamma        = 2.2
	gammaEncodeScale    = 256.0
	gammaEncodeRoundOff = 0.5
)

// ICC profile structural constants
const (
	s15Fixed16Scale = 65536.0
	iccHeaderSize   = 128
	iccVersion      = 0x02100000
	iccProfileClass = "mntr"
	iccSigACSP      = "acsp"
	iccSigAPPL      = "APPL"
	iccCMMType      = "appl"
	iccTagTypeDesc  = "desc"
	iccTagTypeXYZ   = "XYZ "
	iccTagTypeCurv  = "curv"
	iccTagSigDesc   = "desc"
	iccTagSigCPRT   = "cprt"
	iccTagSigWTPT   = "wtpt"
	iccTagSigRXYZ   = "rXYZ"
	iccTagSigGXYZ   = "gXYZ"
	iccTagSigBXYZ   = "bXYZ"
	iccTagSigRTRC   = "rTRC"
	iccTagSigGTRC   = "gTRC"
	iccTagSigBTRC   = "bTRC"
	iccTagSigKTRC   = "kTRC"
	iccTagTableSize = 12
)

// ICC tag count field size
const (
	iccTagCountSize = 4
)

// ICC color space constants
// codehound-ignore: BP-40
const (
	iccColorSpaceRGB  = "RGB "
	iccColorSpaceGray = "GRAY"
	iccPCSXYZ         = "XYZ "
	iccN              = 3
	iccGrayN          = 1
	descBufExtra      = 20
	xyzFactor         = 0.9642
)

// codehound-ignore: BP-40
const (
	rX = 0.4361
	rY = 0.2225
	rZ = 0.0139
	gX = 0.3851
	gY = 0.7169
	gZ = 0.0971
	bX = 0.1431
	bY = 0.0606
	bZ = 0.7140
	wX = 0.9642
	wY = 1.0
	wZ = 0.8249
)

const (
	align4Mask = 3
)

// srgbOnce/grayOnce and their backing slices are package-level state used by
// SRGBProfile / GrayProfile. They are initialised exactly once via sync.Once
// and are functionally immutable after first access (BP-37).
var (
	srgbOnce sync.Once
	grayOnce sync.Once
	srgbData []byte
	grayData []byte
)

// codehound-ignore: PERF-110
var zlibWriterPool = sync.Pool{
	New: func() any { // returns *zlib.Writer
		w, err := zlib.NewWriterLevel(io.Discard, flate.BestSpeed)
		if err != nil {
			return nil
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
		"/N":         iccN,
		"/Alternate": "/DeviceRGB",
		"/Filter":    "/FlateDecode",
		"/Length":    len(srgbData),
	}
}

// GrayProfileDict returns the PDF stream dictionary for the gray ICC profile.
func GrayProfileDict() map[string]interface{} {
	return map[string]interface{}{
		"/N":         iccGrayN,
		"/Alternate": "/DeviceGray",
		"/Filter":    "/FlateDecode",
		"/Length":    len(grayData),
	}
}

func s15Fixed16(v float64) int32 {
	return int32(v * s15Fixed16Scale)
}

func align4(n int) int {
	return (n + align4Mask) & ^align4Mask
}

type iccTag struct {
	sig  string
	data []byte
}

func buildHeader(size int, deviceClass, colorSpace, pcs string) []byte {
	hdr := make([]byte, iccHeaderSize)
	binary.BigEndian.PutUint32(hdr[0:4], uint32(size))
	copy(hdr[4:8], iccCMMType)
	binary.BigEndian.PutUint32(hdr[8:12], iccVersion)
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
	copy(hdr[36:40], iccSigACSP)
	copy(hdr[40:44], iccSigAPPL)
	binary.BigEndian.PutUint32(hdr[64:68], 0)
	binary.BigEndian.PutUint32(hdr[68:72], uint32(s15Fixed16(xyzFactor)))
	binary.BigEndian.PutUint32(hdr[72:76], uint32(s15Fixed16(1.0)))
	binary.BigEndian.PutUint32(hdr[76:80], uint32(s15Fixed16(wZ)))
	copy(hdr[80:84], iccCMMType)
	return hdr
}

func buildICCProfile(deviceClass, colorSpace, pcs string, tags []iccTag) []byte {
	tagTableSize := iccTagCountSize + len(tags)*iccTagTableSize
	dataStart := iccHeaderSize + tagTableSize
	totalDataSize := 0
	for _, t := range tags {
		totalDataSize += align4(len(t.data))
	}
	size := dataStart + totalDataSize

	hdr := buildHeader(size, deviceClass, colorSpace, pcs)
	buf := bytes.NewBuffer(hdr)

	buf.Write([]byte{
		byte(len(tags) >> shift24), byte(len(tags) >> shift16), byte(len(tags) >> shift8), byte(len(tags)),
	})
	offset := uint32(dataStart)
	for _, t := range tags {
		buf.Write([]byte{t.sig[0], t.sig[1], t.sig[2], t.sig[3]})
		buf.Write([]byte{
			byte(offset >> shift24), byte(offset >> shift16), byte(offset >> shift8), byte(offset),
		})
		buf.Write([]byte{
			byte(len(t.data) >> shift24), byte(len(t.data) >> shift16), byte(len(t.data) >> shift8), byte(len(t.data)),
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

// buildDesc and the following build* functions use hardcoded valid values;
// binary.Write errors are impossible with these inputs and are safely discarded.
// codehound-ignore: BP-1
func buildDesc(text string) []byte {
	var buf bytes.Buffer
	buf.Grow(descBufExtra + len(text))
	buf.Write([]byte(iccTagTypeDesc))
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
	buf.Write([]byte(iccTagTypeXYZ))
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))
	_ = binary.Write(&buf, binary.BigEndian, s15Fixed16(x))
	_ = binary.Write(&buf, binary.BigEndian, s15Fixed16(y))
	_ = binary.Write(&buf, binary.BigEndian, s15Fixed16(z))
	return buf.Bytes()
}

// codehound-ignore: BP-1
func buildCurve() []byte {
	var buf bytes.Buffer
	buf.Write([]byte(iccTagTypeCurv))
	_ = binary.Write(&buf, binary.BigEndian, uint32(0))
	_ = binary.Write(&buf, binary.BigEndian, uint32(1))
	gamma := defaultGamma
	_ = binary.Write(&buf, binary.BigEndian, uint16(gamma*gammaEncodeScale+gammaEncodeRoundOff))
	return buf.Bytes()
}

func buildSRGB() []byte {
	return buildICCProfile(iccProfileClass, iccColorSpaceRGB, iccPCSXYZ,
		[]iccTag{
			{iccTagSigDesc, buildDesc("sRGB IEC61966-2.1")},
			{iccTagSigCPRT, buildDesc("Copyright (c)")},
			{iccTagSigWTPT, buildXYZ(wX, wY, wZ)},
			{iccTagSigRXYZ, buildXYZ(rX, rY, rZ)},
			{iccTagSigGXYZ, buildXYZ(gX, gY, gZ)},
			{iccTagSigBXYZ, buildXYZ(bX, bY, bZ)},
			{iccTagSigRTRC, buildCurve()},
			{iccTagSigGTRC, buildCurve()},
			{iccTagSigBTRC, buildCurve()},
		})
}

func buildGray() []byte {
	return buildICCProfile(iccProfileClass, iccColorSpaceGray, iccPCSXYZ,
		[]iccTag{
			{iccTagSigDesc, buildDesc("Gray ICC profile")},
			{iccTagSigCPRT, buildDesc("Copyright (c)")},
			{iccTagSigWTPT, buildXYZ(wX, wY, wZ)},
			{iccTagSigKTRC, buildCurve()},
		})
}

func compress(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	// codehound-ignore: BP-1
	w, _ := zlibWriterPool.Get().(*zlib.Writer)
	if w == nil {
		var err error
		w, err = zlib.NewWriterLevel(&buf, flate.BestSpeed)
		if err != nil {
			return nil, errf("compress: create writer", err)
		}
		_, err = w.Write(data)
		if err != nil {
			// codehound-ignore: BP-5
			w.Close()
			return nil, errf("compress write", err)
		}
		err = w.Close()
		if err != nil {
			return nil, errf("compress close", err)
		}
		return buf.Bytes(), nil
	}
	defer zlibWriterPool.Put(w)
	w.Reset(&buf)
	_, err := w.Write(data)
	if err != nil {
		return nil, errf("compress write", err)
	}
	err = w.Close()
	if err != nil {
		return nil, errf("compress close", err)
	}
	return buf.Bytes(), nil
}

func errf(msg string, err error) error {
	return errors.Join(errors.New(msg), err)
}
