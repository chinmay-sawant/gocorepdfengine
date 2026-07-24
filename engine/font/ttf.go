package font

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"sort"
)

type tableDirEntry struct {
	tag    [4]byte
	check  uint32
	offset uint32
	length uint32
}

type cmapEncodingRecord struct {
	platformID uint16
	encodingID uint16
	offset     uint32
}

type nameRecord struct {
	platformID uint16
	encodingID uint16
	languageID uint16
	nameID     uint16
	length     uint16
	offset     uint16
}

func readU16(data []byte, off uint32) (uint16, uint32) {
	return binary.BigEndian.Uint16(data[off:]), off + 2
}

func readI16(data []byte, off uint32) (int16, uint32) {
	return int16(binary.BigEndian.Uint16(data[off:])), off + 2
}

func readFixed1616(data []byte, off uint32) (float64, uint32) {
	major := binary.BigEndian.Uint16(data[off:])
	fraction := binary.BigEndian.Uint16(data[off+2:])
	return float64(major) + float64(fraction)/65536.0, off + 4
}

func readTag(data []byte, off uint32) ([4]byte, uint32) {
	var tag [4]byte
	copy(tag[:], data[off:off+4])
	return tag, off + 4
}

func findTable(data []byte, tableTag string) ([]byte, error) {
	if len(data) < 12 {
		return nil, errors.New("font: data too short for offset table")
	}
	numTables := binary.BigEndian.Uint16(data[4:])
	dirOffset := uint32(12)
	for i := uint16(0); i < numTables; i++ {
		if uint32(len(data)) < dirOffset+16 {
			return nil, errors.New("font: truncated table directory")
		}
		tag := string(data[dirOffset : dirOffset+4])
		offset := binary.BigEndian.Uint32(data[dirOffset+8:])
		length := binary.BigEndian.Uint32(data[dirOffset+12:])
		if tag == tableTag {
			if uint32(len(data)) < offset+length {
				return nil, fmt.Errorf("font: table %s truncated", tableTag)
			}
			return data[offset : offset+length], nil
		}
		dirOffset += 16
	}
	return nil, fmt.Errorf("font: table %s not found", tableTag)
}

func tableDir(data []byte) ([]tableDirEntry, error) {
	if len(data) < 12 {
		return nil, errors.New("font: data too short")
	}
	numTables := binary.BigEndian.Uint16(data[4:])
	entries := make([]tableDirEntry, numTables)
	dirOffset := uint32(12)
	for i := uint16(0); i < numTables; i++ {
		if uint32(len(data)) < dirOffset+16 {
			return nil, errors.New("font: truncated table directory")
		}
		tag, _ := readTag(data, dirOffset) //nolint: errcheck
		check := binary.BigEndian.Uint32(data[dirOffset+4:])
		offset := binary.BigEndian.Uint32(data[dirOffset+8:])
		length := binary.BigEndian.Uint32(data[dirOffset+12:])
		entries[i] = tableDirEntry{tag: tag, check: check, offset: offset, length: length}
		dirOffset += 16
	}
	return entries, nil
}

// LoadFromPath loads a TTF font from the given file path.
func LoadFromPath(path string) (*Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("font: reading %s: %w", path, err)
	}
	return LoadFromBytes(data)
}

// LoadFromBytes loads a TTF font from raw byte data.
func LoadFromBytes(data []byte) (*Font, error) {
	if len(data) < 12 { // cold path (invalid data)
		return nil, errors.New("font: data too short for TTF header")
	}

	sfVersion := binary.BigEndian.Uint32(data)
	if sfVersion != 0x00010000 && sfVersion != 0x4F54544F {
		return nil, fmt.Errorf("font: not a TTF file (sfVersion=0x%08X)", sfVersion) //nolint: perflint // cold path (one-time validation)
	}

	f := &Font{
		RawData:      data,
		Glyphs:       make(map[rune]*Glyph, 256),       //nolint: perflint // BP-52/PERF-192: size hint for typical glyph count
		cmap:         make(map[rune]uint16),
		glyphMetrics: make(map[uint16]*Glyph, 256),     //nolint: perflint // BP-52/PERF-192: size hint for typical glyph count
	}

	if err := parseHead(f, data); err != nil {
		return nil, err
	}
	if err := parseHHEA(f, data); err != nil {
		return nil, err
	}
	if err := parseHMTX(f, data); err != nil {
		return nil, err
	}
	if err := parseMaxp(f, data); err != nil {
		return nil, err
	}
	if err := parseCMap(f, data); err != nil {
		return nil, err
	}
	if err := parseName(f, data); err != nil {
		return nil, err
	}
	if err := parseOS2(f, data); err != nil {
		return nil, err
	}
	if err := parsePost(f, data); err != nil {
		return nil, err
	}
	if err := parseGlyf(f, data); err != nil {
		return nil, err
	}

	return f, nil
}

func parseHead(f *Font, data []byte) error {
	tbl, err := findTable(data, "head")
	if err != nil {
		return fmt.Errorf("font: parsing head table: %w", err)
	}
	if len(tbl) < 54 {
		return errors.New("font: head table too short")
	}
	off := uint32(0)
	off += 4 // version
	off += 4 // fontRevision
	off += 4 // checkSumAdjustment
	off += 4 // magicNumber
	flags, off := readU16(tbl, off)
	f.UnitsPerEm, off = readU16(tbl, off)
	off += 8 // created
	off += 8 // modified
	f.FontBBox[0], off = readI16(tbl, off) // xMin
	f.FontBBox[1], off = readI16(tbl, off) // yMin
	f.FontBBox[2], off = readI16(tbl, off) // xMax
	f.FontBBox[3], off = readI16(tbl, off) // yMax
	macStyle, _ := readU16(tbl, off) //nolint: errcheck

	f.IsSerif = flags&0x02 != 0
	f.IsMono = macStyle&0x04 != 0
	return nil
}

func parseHHEA(f *Font, data []byte) error {
	tbl, err := findTable(data, "hhea")
	if err != nil {
		return fmt.Errorf("font: parsing hhea table: %w", err)
	}
	if len(tbl) < 36 {
		return errors.New("font: hhea table too short")
	}
	f.Ascent, _ = readI16(tbl, 4) //nolint: errcheck
	f.Descent, _ = readI16(tbl, 6) //nolint: errcheck
	return nil
}

func parseHMTX(_ *Font, data []byte) error {
	_, err := findTable(data, "hmtx")
	if err != nil {
		return fmt.Errorf("font: parsing hmtx table: %w", err)
	}
	return nil
}

func parseMaxp(_ *Font, data []byte) error {
	_, err := findTable(data, "maxp")
	if err != nil {
		return fmt.Errorf("font: parsing maxp table: %w", err)
	}
	return nil
}

func parseCMap(f *Font, data []byte) error {
	tbl, err := findTable(data, "cmap")
	if err != nil {
		return fmt.Errorf("font: parsing cmap table: %w", err)
	}
	if len(tbl) < 4 {
		return errors.New("font: cmap table too short")
	}

	version := binary.BigEndian.Uint16(tbl)
	if version != 0 {
		return fmt.Errorf("font: unsupported cmap version %d", version)
	}

	numTables := binary.BigEndian.Uint16(tbl[2:])
	records := make([]cmapEncodingRecord, numTables)
	for i := uint16(0); i < numTables; i++ {
		base := uint32(4 + i*8)
		if uint32(len(tbl)) < base+8 {
			return errors.New("font: cmap encoding record truncated")
		}
		records[i] = cmapEncodingRecord{
			platformID: binary.BigEndian.Uint16(tbl[base:]),
			encodingID: binary.BigEndian.Uint16(tbl[base+2:]),
			offset:     binary.BigEndian.Uint32(tbl[base+4:]),
		}
	}

	// Prefer format 12, then format 4, then format 0
	type candidate struct {
		format       uint16
		platformID   uint16
		encodingID   uint16
		subtableData []byte
	}
	var candidates []candidate

	for _, rec := range records {
		if uint32(len(tbl)) < rec.offset+2 {
			continue
		}
		format := binary.BigEndian.Uint16(tbl[rec.offset:])
		switch format {
		case 0, 4, 12:
			candidates = append(candidates, candidate{
				format:       format,
				platformID:   rec.platformID,
				encodingID:   rec.encodingID,
				subtableData: tbl[rec.offset:],
			})
		}
	}

	sort.Slice(candidates, func(i, j int) bool {
		pi, pj := candidates[i].platformID, candidates[j].platformID
		if pi != pj {
			return pi < pj
		}
		return candidates[i].format > candidates[j].format
	})

	for _, c := range candidates {
		// Each candidate has unique format/subtableData, so call cannot be hoisted
		ok := parseCMapSubtable(f, c.format, c.subtableData)
		if ok && len(f.cmap) > 0 {
			return nil
		}
	}

	if len(candidates) == 0 {
		return errors.New("font: no supported cmap subtable found")
	}
	return nil
}

func parseCMapSubtable(f *Font, format uint16, data []byte) bool {
	switch format {
	case 0:
		return parseCMapFormat0(f, data)
	case 4:
		return parseCMapFormat4(f, data)
	case 12:
		return parseCMapFormat12(f, data)
	}
	return false
}

func parseCMapFormat0(f *Font, data []byte) bool {
	if len(data) < 262 {
		return false
	}
	for i := uint16(0); i < 256; i++ {
		gid := data[6+i]
		if gid != 0 {
			f.cmap[rune(i)] = uint16(gid)
		}
	}
	return true
}

func parseCMapFormat4(f *Font, data []byte) bool {
	if len(data) < 14 {
		return false
	}
	length := binary.BigEndian.Uint16(data[2:])
	if uint32(len(data)) < uint32(length) {
		return false
	}
	segCountX2 := binary.BigEndian.Uint16(data[6:])
	segCount := segCountX2 / 2

	endCodes := make([]uint16, segCount)
	startCodes := make([]uint16, segCount)
	idDeltas := make([]int16, segCount)
	idRangeOffsets := make([]uint16, segCount)

	base := uint32(14)
	for i := uint16(0); i < segCount; i++ {
		off := base + uint32(i*2)
		endCodes[i] = binary.BigEndian.Uint16(data[off:])
	}
	for i := uint16(0); i < segCount; i++ {
		off := base + 2 + uint32(segCount*2) + uint32(i*2)
		startCodes[i] = binary.BigEndian.Uint16(data[off:])
	}

	for i := uint16(0); i < segCount; i++ {
		off := base + 2 + uint32(segCount*4) + uint32(i*2)
		idDeltas[i] = int16(binary.BigEndian.Uint16(data[off:]))
	}

	for i := uint16(0); i < segCount; i++ {
		off := base + 2 + uint32(segCount*6) + uint32(i*2)
		idRangeOffsets[i] = binary.BigEndian.Uint16(data[off:])
	}

	for i := uint16(0); i < segCount; i++ {
		if startCodes[i] == 0xFFFF || endCodes[i] == 0xFFFF {
			continue
		}
		for c := startCodes[i]; c <= endCodes[i]; c++ {
			var gid uint16
			if idRangeOffsets[i] == 0 {
				gid = uint16(int32(c) + int32(idDeltas[i])) & 0xFFFF
			} else {
				rangeOff := uint32(idRangeOffsets[i]) + uint32((c-startCodes[i])*2)
				actualOff := base + 2 + uint32(segCount*6) + uint32(i*2) + rangeOff
				if uint32(len(data)) < actualOff+2 {
					continue
				}
				gid = binary.BigEndian.Uint16(data[actualOff:])
				if gid != 0 {
					gid = uint16(int32(gid) + int32(idDeltas[i])) & 0xFFFF
				}
			}
			if gid != 0 {
				f.cmap[rune(c)] = gid
			}
		}
	}
	return true
}

func parseCMapFormat12(f *Font, data []byte) bool {
	if len(data) < 16 {
		return false
	}
	nGroups := binary.BigEndian.Uint32(data[12:])
	base := uint32(16)
	for i := uint32(0); i < nGroups; i++ {
		off := base + i*12
		if uint32(len(data)) < off+12 {
			return false
		}
		startCode := binary.BigEndian.Uint32(data[off:])
		endCode := binary.BigEndian.Uint32(data[off+4:])
		startGID := binary.BigEndian.Uint32(data[off+8:])
		for c := startCode; c <= endCode; c++ {
			gid := uint16(startGID + (c - startCode))
			if gid != 0 {
				f.cmap[rune(c)] = gid
			}
		}
	}
	return true
}

func parseName(f *Font, data []byte) error {
	tbl, err := findTable(data, "name")
	if err != nil {
		return fmt.Errorf("font: name table not found: %w", err)
	}
	if len(tbl) < 6 {
		return errors.New("font: name table too short")
	}

	count := binary.BigEndian.Uint16(tbl[2:])
	storage := binary.BigEndian.Uint16(tbl[4:])

	nameIDs := map[uint16]*string{}
	nameIDs[1] = &f.Family
	nameIDs[2] = &f.Style
	nameIDs[6] = &f.Name

	for i := uint16(0); i < count; i++ {
		base := uint32(6 + i*12)
		if uint32(len(tbl)) < base+12 {
			continue
		}
		nr := nameRecord{
			platformID: binary.BigEndian.Uint16(tbl[base:]),
			encodingID: binary.BigEndian.Uint16(tbl[base+2:]),
			languageID: binary.BigEndian.Uint16(tbl[base+4:]),
			nameID:     binary.BigEndian.Uint16(tbl[base+6:]),
			length:     binary.BigEndian.Uint16(tbl[base+8:]),
			offset:     binary.BigEndian.Uint16(tbl[base+10:]),
		}

		target := nameIDs[nr.nameID]
		if target == nil {
			continue
		}
		if *target != "" {
			continue
		}

		strOff := uint32(storage) + uint32(nr.offset)
		if uint32(len(tbl)) < strOff+uint32(nr.length) {
			continue
		}

		raw := tbl[strOff : strOff+uint32(nr.length)]

		if nr.platformID == 0 || nr.platformID == 3 {
			// Unicode (UCS-2 or UTF-16BE)
			chars := make([]rune, 0, nr.length/2)
			for i := uint16(0); i+1 < nr.length; i += 2 {
				u := binary.BigEndian.Uint16(raw[i:])
				chars = append(chars, rune(u))
			}
			*target = string(chars)
		} else if nr.platformID == 1 {
			// Mac Roman
			*target = string(raw)
		}
	}

	return nil
}

func parseOS2(f *Font, data []byte) error {
	tbl, err := findTable(data, "OS/2")
	if err != nil {
		return fmt.Errorf("font: OS/2 table not found: %w", err)
	}
	if len(tbl) < 86 {
		return errors.New("font: OS/2 table too short")
	}

	version := binary.BigEndian.Uint16(tbl)
	_ = version

	fsSelection := binary.BigEndian.Uint16(tbl[62:])

	f.Flags |= 1 << 2

	if fsSelection&0x01 != 0 {
		f.Flags |= 1 << 0
	}
	if fsSelection&0x08 != 0 {
		f.Flags |= 1 << 1
	}
	if fsSelection&0x20 != 0 {
		f.Flags |= 1 << 5
	}

	// sCapHeight and sxHeight only exist in OS/2 v2+
	if version >= 2 && len(tbl) >= 90 {
		f.XHeight = int16(binary.BigEndian.Uint16(tbl[86:]))
		f.CapHeight = int16(binary.BigEndian.Uint16(tbl[88:]))
	}

	if f.CapHeight < 0 {
		f.CapHeight = 0
	}
	if f.XHeight < 0 {
		f.XHeight = 0
	}

	return nil
}

func parsePost(f *Font, data []byte) error {
	tbl, err := findTable(data, "post")
	if err != nil {
		return fmt.Errorf("font: post table not found: %w", err)
	}
	if len(tbl) < 32 {
		return errors.New("font: post table too short")
	}

	off := uint32(0)
	off += 4 // version
	f.ItalicAngle, off = readFixed1616(tbl, off)
	off += 2 // underlinePosition
	off += 2 // underlineThickness
	off += 4 // isFixedPitch
	_ = off
	_ = f.ItalicAngle

	return nil
}

func parseGlyf(f *Font, data []byte) error {
	tbl, err := findTable(data, "glyf")
	if err != nil {
		return fmt.Errorf("font: glyf table not found: %w", err)
	}

	locaTable, err := findTable(data, "loca")
	if err != nil {
		return fmt.Errorf("font: loca table not found: %w", err)
	}

	headTable, err := findTable(data, "head")
	if err != nil {
		return fmt.Errorf("font: parsing glyf table: finding head: %w", err)
	}

	indexToLocFormat := binary.BigEndian.Uint16(headTable[50:])

	maxpTable, err := findTable(data, "maxp")
	if err != nil {
		return fmt.Errorf("font: maxp table not found: %w", err)
	}
	numGlyphs := binary.BigEndian.Uint16(maxpTable[4:])

	hmtxTable, err := findTable(data, "hmtx")
	if err != nil {
		return fmt.Errorf("font: hmtx table not found: %w", err)
	}

	// Determine number of hmetrics
	var numHMetrics uint16
	if hheaTable, err := findTable(data, "hhea"); err == nil && len(hheaTable) >= 36 {
		numHMetrics = binary.BigEndian.Uint16(hheaTable[34:])
	} else {
		numHMetrics = numGlyphs
	}

	// Build glyph metrics for all GIDs up to numGlyphs
	for gid := uint16(0); gid < numGlyphs; gid++ {
		g := &Glyph{GID: gid}

		// Get advance width from hmtx
		var width int16
		if gid < numHMetrics {
			width = int16(binary.BigEndian.Uint16(hmtxTable[uint32(gid)*4:]))
		} else if numHMetrics > 0 {
			width = int16(binary.BigEndian.Uint16(hmtxTable[uint32(numHMetrics-1)*4:]))
		}
		g.Width = width

		// Get glyph bounding box from glyf/loca
		var glyphOffset uint32
		var glyphLength uint32
		if indexToLocFormat == 0 {
			off := uint32(gid) * 2
			if uint32(len(locaTable)) >= off+4 {
				glyphOffset = uint32(binary.BigEndian.Uint16(locaTable[off:])) * 2
				glyphLength = uint32(binary.BigEndian.Uint16(locaTable[off+2:]))*2 - glyphOffset
			}
		} else {
			off := uint32(gid) * 4
			if uint32(len(locaTable)) >= off+8 {
				glyphOffset = binary.BigEndian.Uint32(locaTable[off:])
				glyphLength = binary.BigEndian.Uint32(locaTable[off+4:]) - glyphOffset
			}
		}

		if glyphLength >= 12 && uint32(len(tbl)) >= glyphOffset+12 {
			g.BBox = [4]int16{
				int16(binary.BigEndian.Uint16(tbl[glyphOffset+2:])),
				int16(binary.BigEndian.Uint16(tbl[glyphOffset+4:])),
				int16(binary.BigEndian.Uint16(tbl[glyphOffset+6:])),
				int16(binary.BigEndian.Uint16(tbl[glyphOffset+8:])),
			}
		}

		f.glyphMetrics[gid] = g
	}

	return nil
}

// IsTTF checks whether the given data represents a TTF font.
func IsTTF(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	sfVersion := binary.BigEndian.Uint32(data)
	return sfVersion == 0x00010000 || sfVersion == 0x4F54544F
}

// StemVValue returns the StemV value for the font descriptor.
func (f *Font) StemVValue() int16 {
	if f.StemV != 0 {
		return f.StemV
	}
	weight := 400
	if f.Style == "Bold" || f.Style == "BoldItalic" {
		weight = 700
	}
	if weight < 500 {
		return 50
	} else if weight < 700 {
		return 70
	}
	return 85
}
