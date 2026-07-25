package font

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

// AddChar adds a single rune to the font subset.
func (f *Font) AddChar(r rune) {
	if f.Glyphs == nil {
		f.Glyphs = make(map[rune]*Glyph)
	}
	if _, ok := f.Glyphs[r]; ok {
		return
	}
	gid, ok := f.cmap[r]
	if !ok {
		// Character not in font — use the .notdef glyph (GID 0).
		gid = 0
	}
	if gm, exists := f.glyphMetrics[gid]; exists {
		f.Glyphs[r] = &Glyph{GID: gid, Width: gm.Width, BBox: gm.BBox}
	} else {
		f.Glyphs[r] = &Glyph{GID: gid}
	}
}

// AddChars adds runes to the font subset.
func (f *Font) AddChars(chars []rune) {
	for _, r := range chars {
		f.AddChar(r)
	}
}

// UsedChars returns the runes that have been added to the font subset.
func (f *Font) UsedChars() []rune {
	return f.UsedRunes()
}

// GenerateSubset generates a subset TTF from the font raw data containing only the used glyphs.
func (f *Font) GenerateSubset() error {
	if len(f.RawData) < 12 {
		return errors.New("font: no raw data to subset")
	}

	usedGIDs := buildUsedGIDSet(f)
	usedGIDs[0] = true

	if len(usedGIDs) <= 1 {
		for _, g := range f.Glyphs {
			usedGIDs[g.GID] = true
		}
	}

	entries, err := tableDir(f.RawData)
	if err != nil { // cold path (error on subset)
		return fmt.Errorf("font: tableDir: %w", err)
	}

	requiredTags := map[string]bool{
		"head": true, "hhea": true, "hmtx": true,
		"maxp": true, "glyf": true, "loca": true,
		"cmap": true, "name": true, "OS/2": true,
		"post": true, "cvt ": true, "prep": true,
		"fpgm": true, "cvt": true,
	}

	subset, err := buildSubsetTTF(f, f.RawData, entries, usedGIDs, requiredTags)
	if err != nil {
		return fmt.Errorf("font: subset generation error: %w", err)
	}

	if len(subset) < 12 {
		return errors.New("font: generated subset is invalid (too short)")
	}

	f.SubsetData = subset
	return nil
}

func buildUsedGIDSet(f *Font) map[uint16]bool {
	used := make(map[uint16]bool, len(f.Glyphs))
	for _, g := range f.Glyphs {
		used[g.GID] = true
	}
	return used
}

func pad4(n uint32) uint32 {
	return (n + 3) & ^uint32(3)
}

func buildSubsetTTF(f *Font, orig []byte, _ []tableDirEntry, usedGIDs map[uint16]bool, _ map[string]bool) ([]byte, error) {
	gidList := make([]uint16, 0, len(usedGIDs))
	for gid := range usedGIDs {
		gidList = append(gidList, gid)
	}
	sort.Slice(gidList, func(i, j int) bool { return gidList[i] < gidList[j] })

	maxpData, _ := findTable(orig, "maxp")
	oldNumGlyphs := uint16(0)
	if len(maxpData) >= 6 {
		oldNumGlyphs = binary.BigEndian.Uint16(maxpData[4:])
	}

	headData, _ := findTable(orig, "head")
	locaFormat := uint16(0)
	if len(headData) >= 52 {
		locaFormat = binary.BigEndian.Uint16(headData[50:])
	}

	hheaData, _ := findTable(orig, "hhea")
	numHMetrics := oldNumGlyphs
	if len(hheaData) >= 36 {
		numHMetrics = binary.BigEndian.Uint16(hheaData[34:])
	}

	glyfTable, _ := findTable(orig, "glyf")
	locaTable, _ := findTable(orig, "loca")
	hmtxTable, _ := findTable(orig, "hmtx")

	type glyphEntry struct {
		gid    uint16
		data   []byte
		length uint32
		width  uint16
	}
	var glyphs []glyphEntry

	gidMap := make(map[uint16]uint16, len(gidList))
	for newID, oldGID := range gidList {
		gidMap[oldGID] = uint16(newID)
	}
	f.SubGIDMap = gidMap

	type glyphInfo struct {
		gid    uint16
		offset uint32
		length uint32
		width  uint16
	}
	infos := make([]glyphInfo, 0, len(gidList))
	var totalGlyphSize uint32

	for _, oldGID := range gidList {
		var length uint32
		var glyphOffset uint32

		if glyfTable != nil && locaTable != nil {
			var nextOffset uint32
			if locaFormat == 0 {
				off := uint32(oldGID) * 2
				if uint32(len(locaTable)) >= off+4 {
					glyphOffset = uint32(binary.BigEndian.Uint16(locaTable[off:])) * 2
					nextOffset = uint32(binary.BigEndian.Uint16(locaTable[off+2:])) * 2
				}
			} else {
				off := uint32(oldGID) * 4
				if uint32(len(locaTable)) >= off+8 {
					glyphOffset = binary.BigEndian.Uint32(locaTable[off:])
					nextOffset = binary.BigEndian.Uint32(locaTable[off+4:])
				}
			}
			length = nextOffset - glyphOffset
			if length > 0 && uint32(len(glyfTable)) >= glyphOffset+length {
				totalGlyphSize += pad4(length)
			} else {
				length = 0
			}
		}

		var width uint16
		if hmtxTable != nil {
			if oldGID < numHMetrics {
				width = binary.BigEndian.Uint16(hmtxTable[uint32(oldGID)*4:])
			} else if numHMetrics > 0 {
				width = binary.BigEndian.Uint16(hmtxTable[uint32(numHMetrics-1)*4:])
			}
		}

		infos = append(infos, glyphInfo{
			gid: oldGID, offset: glyphOffset, length: length, width: width,
		})
	}

	glyphDataBuf := make([]byte, totalGlyphSize)
	var dataOffset uint32
	for _, info := range infos {
		var data []byte
		if info.length > 0 {
			data = glyphDataBuf[dataOffset : dataOffset+info.length]
			copy(data, glyfTable[info.offset:info.offset+info.length])
		}
		glyphs = append(glyphs, glyphEntry{
			gid: info.gid, data: data, length: info.length, width: info.width,
		})
		dataOffset += pad4(info.length)
	}

	newNumGlyphs := uint16(len(glyphs))

	var newLocaData []byte
	if locaFormat == 0 {
		var glyphOffset uint32
		for _, ge := range glyphs {
			v := uint16(glyphOffset / 2)
			newLocaData = append(newLocaData, byte(v>>8), byte(v))
			glyphOffset += pad4(ge.length)
			v = uint16(glyphOffset / 2)
			newLocaData = append(newLocaData, byte(v>>8), byte(v))
		}
	} else {
		var glyphOffset uint32
		for _, ge := range glyphs {
			newLocaData = append(newLocaData,
				byte(glyphOffset>>24), byte(glyphOffset>>16),
				byte(glyphOffset>>8), byte(glyphOffset))
			glyphOffset += pad4(ge.length)
			newLocaData = append(newLocaData,
				byte(glyphOffset>>24), byte(glyphOffset>>16),
				byte(glyphOffset>>8), byte(glyphOffset))
		}
	}

	var newGlyfData []byte
	for _, ge := range glyphs {
		newGlyfData = append(newGlyfData, ge.data...)
		padLen := pad4(ge.length) - ge.length
		for j := uint32(0); j < padLen; j++ {
			newGlyfData = append(newGlyfData, 0)
		}
	}

	newHmtxData := make([]byte, 0, len(glyphs)*4)
	for _, ge := range glyphs {
		newHmtxData = append(newHmtxData, byte(ge.width>>8), byte(ge.width), 0, 0)
	}

	var newMaxpData []byte
	if len(maxpData) >= 6 {
		newMaxpData = make([]byte, len(maxpData))
		copy(newMaxpData, maxpData)
		newMaxpData[4] = byte(newNumGlyphs >> 8)
		newMaxpData[5] = byte(newNumGlyphs)
	} else {
		newMaxpData = make([]byte, 32)
		newMaxpData[0] = 0
		newMaxpData[1] = 0
		newMaxpData[2] = 1
		newMaxpData[3] = 0
		newMaxpData[4] = byte(newNumGlyphs >> 8)
		newMaxpData[5] = byte(newNumGlyphs)
		newMaxpData[6] = byte(newNumGlyphs >> 8)
		newMaxpData[7] = byte(newNumGlyphs)
	}

	var newHeadData []byte
	if len(headData) >= 54 {
		newHeadData = make([]byte, len(headData))
		copy(newHeadData, headData)
		newHeadData[50] = byte(locaFormat >> 8)
		newHeadData[51] = byte(locaFormat)
		newHeadData[8] = 0
		newHeadData[9] = 0
		newHeadData[10] = 0
		newHeadData[11] = 0
	} else {
		return nil, errors.New("font: head table missing")
	}

	var newHheaData []byte
	if len(hheaData) >= 36 {
		newHheaData = make([]byte, len(hheaData))
		copy(newHheaData, hheaData)
		newHheaData[34] = byte(newNumGlyphs >> 8)
		newHheaData[35] = byte(newNumGlyphs)
	} else {
		return nil, errors.New("font: hhea table missing")
	}

	var newCMapData []byte
	{
		usedChars := f.UsedChars()
		subtable := buildFormat4CMap(usedChars, gidMap)
		headerLen := uint32(4 + 8)
		cmap := make([]byte, headerLen+uint32(len(subtable)))
		cmap[0] = 0
		cmap[1] = 0
		cmap[2] = 0
		cmap[3] = 1
		cmap[4] = 0
		cmap[5] = 3
		cmap[6] = 0
		cmap[7] = 1
		cmap[8] = byte(headerLen >> 24)
		cmap[9] = byte(headerLen >> 16)
		cmap[10] = byte(headerLen >> 8)
		cmap[11] = byte(headerLen)
		copy(cmap[headerLen:], subtable)
		newCMapData = cmap
	}

	findTableData := func(tag string) []byte {
		d, err := findTable(orig, tag)
		if err != nil {
			return nil
		}
		return d
	}

	type tbl struct {
		tag  string
		data []byte
	}

	var tables []tbl
	addTable := func(tag string, data []byte) {
		if data != nil {
			tables = append(tables, tbl{tag: tag, data: data})
		}
	}

	addTable("head", newHeadData)
	addTable("hhea", newHheaData)
	addTable("maxp", newMaxpData)
	addTable("OS/2", findTableData("OS/2"))
	addTable("name", findTableData("name"))
	addTable("cmap", newCMapData)
	postData := findTableData("post")
	if len(postData) >= 34 {
		newPostData := make([]byte, len(postData))
		copy(newPostData, postData)
		newPostData[32] = byte(newNumGlyphs >> 8)
		newPostData[33] = byte(newNumGlyphs)
		postData = newPostData
	}
	addTable("post", postData)
	addTable("loca", newLocaData)
	addTable("glyf", newGlyfData)
	addTable("hmtx", newHmtxData)

	cvt := findTableData("cvt ")
	if cvt == nil {
		cvt = findTableData("cvt")
	}
	addTable("prep", findTableData("prep"))
	addTable("fpgm", findTableData("fpgm"))
	addTable("cvt ", cvt)

	numTables := uint16(len(tables))

	entrySelector := uint16(0)
	for 1<<(entrySelector+1) <= numTables {
		entrySelector++
	}
	searchRange := uint16(1 << entrySelector) * 16
	rangeShift := numTables*16 - searchRange

	header := make([]byte, 12)
	header[0] = 0
	header[1] = 0
	header[2] = 1
	header[3] = 0
	header[4] = byte(numTables >> 8)
	header[5] = byte(numTables)
	header[6] = byte(searchRange >> 8)
	header[7] = byte(searchRange)
	header[8] = byte(entrySelector >> 8)
	header[9] = byte(entrySelector)
	header[10] = byte(rangeShift >> 8)
	header[11] = byte(rangeShift)

	type tableInfo struct {
		tag    string
		data   []byte
		offset uint32
		length uint32
	}

	var tableInfos []tableInfo
	off := uint32(12) + uint32(numTables)*16
	for _, t := range tables {
		length := uint32(len(t.data))
		tableInfos = append(tableInfos, tableInfo{
			tag: t.tag, data: t.data, offset: off, length: length,
		})
		off += pad4(length)
	}

	out := make([]byte, off)
	copy(out, header)

	dirBase := uint32(12)
	for _, ti := range tableInfos {
		copy(out[dirBase:], ti.tag)
		out[dirBase+8] = byte(ti.offset >> 24)
		out[dirBase+9] = byte(ti.offset >> 16)
		out[dirBase+10] = byte(ti.offset >> 8)
		out[dirBase+11] = byte(ti.offset)
		out[dirBase+12] = byte(ti.length >> 24)
		out[dirBase+13] = byte(ti.length >> 16)
		out[dirBase+14] = byte(ti.length >> 8)
		out[dirBase+15] = byte(ti.length)
		dirBase += 16
	}

	for _, ti := range tableInfos {
		copy(out[ti.offset:], ti.data)
	}

	// Calculate and set checkSumAdjustment
	checksum := calcFileChecksum(out)
	adjustment := uint32(0xB1B0AFBA) - checksum
	headOff := findTableOffset(out, "head")
	if headOff != 0 {
		out[headOff+8] = byte(adjustment >> 24)
		out[headOff+9] = byte(adjustment >> 16)
		out[headOff+10] = byte(adjustment >> 8)
		out[headOff+11] = byte(adjustment)
	}

	return out, nil
}

func calcFileChecksum(data []byte) uint32 {
	var sum uint32
	for i := 0; i+3 < len(data); i += 4 {
		sum += binary.BigEndian.Uint32(data[i:])
	}
	return sum
}

func findTableOffset(data []byte, tag string) uint32 {
	if len(data) < 12 {
		return 0
	}
	numTables := binary.BigEndian.Uint16(data[4:])
	for i := uint16(0); i < numTables; i++ {
		base := uint32(12) + uint32(i)*16
		if string(data[base:base+4]) == tag {
			return binary.BigEndian.Uint32(data[base+8:])
		}
	}
	return 0
}

func buildFormat4CMap(usedChars []rune, gidMap map[uint16]uint16) []byte {
	sorted := make([]uint16, 0, len(usedChars))
	for _, r := range usedChars {
		if r >= 0 && r <= 0xFFFF {
			sorted = append(sorted, uint16(r))
		}
	}
	sort.Slice(sorted, func(i, j int) bool { return sorted[i] < sorted[j] })

	if len(sorted) == 0 {
		return buildEmptyFormat4CMap()
	}

	type segRange struct {
		start uint16
		end   uint16
	}
	var ranges []segRange

	prev := sorted[0]
	rangeStart := prev
	for i := 1; i < len(sorted); i++ {
		curr := sorted[i]
		if curr == prev+1 {
			prev = curr
			continue
		}
		ranges = append(ranges, segRange{start: rangeStart, end: prev})
		rangeStart = curr
		prev = curr
	}
	ranges = append(ranges, segRange{start: rangeStart, end: prev})

	segCount := len(ranges) + 1

	endCodes := make([]uint16, segCount)
	startCodes := make([]uint16, segCount)
	idDeltas := make([]int16, segCount)

	for i, r := range ranges {
		endCodes[i] = r.end
		startCodes[i] = r.start
		firstNewGID := gidMap[r.start]
		idDeltas[i] = int16(firstNewGID) - int16(r.start)
	}

	endCodes[segCount-1] = 0xFFFF
	startCodes[segCount-1] = 0xFFFF
	idDeltas[segCount-1] = 1

	var glyphIDArray []uint16
	for _, r := range sorted {
		glyphIDArray = append(glyphIDArray, gidMap[r])
	}

	headerLen := uint16(14)
	endCodesLen := uint16(segCount) * 2
	startCodesLen := uint16(segCount) * 2
	deltasLen := uint16(segCount) * 2
	rangeOffsetsLen := uint16(segCount) * 2
	glyphArrayLen := uint16(len(glyphIDArray)) * 2

	totalLength := headerLen + endCodesLen + 2 + startCodesLen + deltasLen + rangeOffsetsLen + glyphArrayLen

	data := make([]byte, totalLength)

	data[0] = 0
	data[1] = 4
	data[2] = byte(totalLength >> 8)
	data[3] = byte(totalLength)
	data[4] = 0
	data[5] = 0
	segCountX2 := uint16(segCount) * 2
	data[6] = byte(segCountX2 >> 8)
	data[7] = byte(segCountX2)

	srEntrySelector := uint16(0)
	for 1<<(srEntrySelector+1) <= uint16(segCount) {
		srEntrySelector++
	}
	srSearchRangeBytes := uint16(1 << srEntrySelector) * 2
	data[8] = byte(srSearchRangeBytes >> 8)
	data[9] = byte(srSearchRangeBytes)
	data[10] = byte(srEntrySelector >> 8)
	data[11] = byte(srEntrySelector)
	v := uint16(segCount)*2 - srSearchRangeBytes
	data[12] = byte(v >> 8)
	data[13] = byte(v)

	off := uint16(14)
	for i := 0; i < segCount; i++ {
		data[off] = byte(endCodes[i] >> 8)
		data[off+1] = byte(endCodes[i])
		off += 2
	}
	data[off] = 0
	data[off+1] = 0
	off += 2
	for i := 0; i < segCount; i++ {
		data[off] = byte(startCodes[i] >> 8)
		data[off+1] = byte(startCodes[i])
		off += 2
	}
	for i := 0; i < segCount; i++ {
		data[off] = byte(uint16(idDeltas[i]) >> 8)
		data[off+1] = byte(uint16(idDeltas[i]))
		off += 2
	}

	for i := 0; i < segCount; i++ {
		data[off] = 0
		data[off+1] = 0
		off += 2
	}

	for _, gid := range glyphIDArray {
		data[off] = byte(gid >> 8)
		data[off+1] = byte(gid)
		off += 2
	}

	return data
}

func buildEmptyFormat4CMap() []byte {
	segCount := uint16(1)
	segCountX2 := segCount * 2
	totalLength := uint16(14 + 2 + 2 + 2 + 2 + 0)
	data := make([]byte, totalLength)
	data[0] = 0
	data[1] = 4
	data[2] = byte(totalLength >> 8)
	data[3] = byte(totalLength)
	data[4] = 0
	data[5] = 0
	data[6] = byte(segCountX2 >> 8)
	data[7] = byte(segCountX2)
	data[8] = 0
	data[9] = 2
	data[10] = 0
	data[11] = 0
	data[12] = 0
	data[13] = 0
	data[14] = 0xFF
	data[15] = 0xFF
	data[16] = 0
	data[17] = 0
	data[18] = 0xFF
	data[19] = 0xFF
	data[20] = 0
	data[21] = 1
	data[22] = 0
	data[23] = 0
	return data
}

