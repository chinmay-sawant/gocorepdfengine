// codehound-ignore-file: BP-1
package font

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
)

const tableCVT = "cvt "

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
	if len(f.RawData) < ttfDirOffset {
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
		// codehound-ignore: PERF-35
		return fmt.Errorf("font: tableDir: %w", err)
	}

	requiredTags := map[string]bool{
		"head": true, "hhea": true, "hmtx": true,
		"maxp": true, "glyf": true, "loca": true,
		"cmap": true, "name": true, "OS/2": true,
		"post": true, "prep": true,
		"fpgm": true, tableCVT: true, "cvt": true,
	}

	subset, err := buildSubsetTTF(f, f.RawData, entries, usedGIDs, requiredTags)
	if err != nil {
		// codehound-ignore: PERF-35
		return fmt.Errorf("font: subset generation error: %w", err)
	}

	if len(subset) < ttfDirOffset {
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
	return (n + pad4Mask) & ^uint32(pad4Mask)
}

type glyphEntry struct {
	gid    uint16
	data   []byte
	length uint32
	width  uint16
}

type glyphInfo struct {
	gid    uint16
	offset uint32
	length uint32
	width  uint16
}

func readGlyphOffsetLength(oldGID uint16, locaTable []byte, locaFormat uint16) (uint32, uint32) {
	if locaFormat == 0 {
		off := uint32(oldGID) * ttfWordSize
		if uint32(len(locaTable)) >= off+4 {
			glyphOffset := uint32(binary.BigEndian.Uint16(locaTable[off:])) * ttfWordSize
			nextOffset := uint32(binary.BigEndian.Uint16(locaTable[off+ttfWordSize:])) * ttfWordSize
			return glyphOffset, nextOffset - glyphOffset
		}
	} else {
		off := uint32(oldGID) * ttfDWordSize
		if uint32(len(locaTable)) >= off+8 {
			glyphOffset := binary.BigEndian.Uint32(locaTable[off:])
			nextOffset := binary.BigEndian.Uint32(locaTable[off+4:])
			return glyphOffset, nextOffset - glyphOffset
		}
	}
	return 0, 0
}

func collectGlyphInfos(gidList []uint16, glyfTable, locaTable, hmtxTable []byte, locaFormat uint16, numHMetrics uint16, f *Font) ([]glyphEntry, map[uint16]uint16) {
	gidMap := make(map[uint16]uint16, len(gidList))
	for newID, oldGID := range gidList {
		gidMap[oldGID] = uint16(newID)
	}
	f.SubGIDMap = gidMap

	infos := make([]glyphInfo, 0, len(gidList))
	var totalGlyphSize uint32

	for _, oldGID := range gidList {
		var length uint32
		var glyphOffset uint32

		if glyfTable != nil && locaTable != nil {
			glyphOffset, length = readGlyphOffsetLength(oldGID, locaTable, locaFormat)
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
	var glyphs = make([]glyphEntry, 0, len(infos))
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

	return glyphs, gidMap
}

func buildGlyphTables(glyphs []glyphEntry, locaFormat uint16) ([]byte, []byte, []byte) {
	var newLocaData []byte
	if locaFormat == 0 {
		var glyphOffset uint32
		for _, ge := range glyphs {
			v := uint16(glyphOffset / ttfWordSize)
			newLocaData = append(newLocaData, byte(v>>shift8), byte(v))
			glyphOffset += pad4(ge.length)
			v = uint16(glyphOffset / ttfWordSize)
			newLocaData = append(newLocaData, byte(v>>shift8), byte(v))
		}
	} else {
		var glyphOffset uint32
		for _, ge := range glyphs {
			newLocaData = append(newLocaData,
				byte(glyphOffset>>shift24), byte(glyphOffset>>shift16),
				byte(glyphOffset>>shift8), byte(glyphOffset))
			glyphOffset += pad4(ge.length)
			newLocaData = append(newLocaData,
				byte(glyphOffset>>shift24), byte(glyphOffset>>shift16),
				byte(glyphOffset>>shift8), byte(glyphOffset))
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

	// codehound-ignore: PERF-240
	newHmtxData := make([]byte, 0, len(glyphs)*hmtxEntrySize)
	for _, ge := range glyphs {
		newHmtxData = append(newHmtxData, byte(ge.width>>shift8), byte(ge.width), 0, 0)
	}

	return newLocaData, newGlyfData, newHmtxData
}

func buildSubsetCMap(f *Font, gidMap map[uint16]uint16) []byte {
	usedChars := f.UsedChars()
	subtable := buildFormat4CMap(usedChars, gidMap)
	headerLen := uint32(cmapHeaderLen + cmapEncRecSize)
	cmap := make([]byte, headerLen+uint32(len(subtable)))
	cmap[0] = 0
	cmap[1] = 0
	cmap[ttfWordSize] = 0
	cmap[ttfWordSize+1] = 1
	cmap[ttfDWordSize] = 0
	cmap[ttfDWordSize+1] = 3
	cmap[ttfMaxpMinLen] = 0
	cmap[ttfMaxpMinLen+1] = 1
	cmap[cmapEncRecSize] = byte(headerLen >> shift24)
	cmap[cmapEncRecSize+1] = byte(headerLen >> shift16)
	cmap[cmapEncRecSize+ttfWordSize] = byte(headerLen >> shift8)
	cmap[cmapEncRecSize+ttfWordSize+1] = byte(headerLen)
	copy(cmap[headerLen:], subtable)
	return cmap
}

func buildTTFHeader(numTables uint16) []byte {
	entrySelector := uint16(0)
	for 1<<(entrySelector+1) <= numTables {
		entrySelector++
	}
	searchRange := uint16(1<<entrySelector) * ttfEntrySize
	rangeShift := numTables*ttfEntrySize - searchRange

	header := make([]byte, ttfDirOffset)
	header[0] = 0
	header[1] = 0
	header[2] = 1
	header[3] = 0
	header[ttfDWordSize] = byte(numTables >> shift8)
	header[ttfDWordSize+1] = byte(numTables)
	header[ttfMaxpMinLen] = byte(searchRange >> shift8)
	header[ttfMaxpMinLen+1] = byte(searchRange)
	header[8] = byte(entrySelector >> shift8)
	header[9] = byte(entrySelector)
	header[10] = byte(rangeShift >> shift8)
	header[11] = byte(rangeShift)
	return header
}

// codehound-ignore: BP-1
func buildSubsetTTF(f *Font, orig []byte, _ []tableDirEntry, usedGIDs map[uint16]bool, _ map[string]bool) ([]byte, error) {
	gidList := make([]uint16, 0, len(usedGIDs))
	for gid := range usedGIDs {
		gidList = append(gidList, gid)
	}
	sort.Slice(gidList, func(i, j int) bool { return gidList[i] < gidList[j] })

	maxpData, _ := findTable(orig, "maxp")
	oldNumGlyphs := uint16(0)
	if len(maxpData) >= ttfMaxpMinLen {
		oldNumGlyphs = binary.BigEndian.Uint16(maxpData[ttfDWordSize:])
	}

	headData, _ := findTable(orig, "head")
	locaFormat := uint16(0)
	if len(headData) >= headLocaFmtOff+ttfWordSize {
		locaFormat = binary.BigEndian.Uint16(headData[headLocaFmtOff:])
	}

	hheaData, _ := findTable(orig, "hhea")
	numHMetrics := oldNumGlyphs
	if len(hheaData) >= ttfHheaMinLen {
		numHMetrics = binary.BigEndian.Uint16(hheaData[ttfHheaMinLen-ttfWordSize:])
	}

	// findTable error discarded; table presence is validated during parsing.
	glyfTable, _ := findTable(orig, "glyf")
	locaTable, _ := findTable(orig, "loca")
	hmtxTable, _ := findTable(orig, "hmtx")

	glyphs, gidMap := collectGlyphInfos(gidList, glyfTable, locaTable, hmtxTable, locaFormat, numHMetrics, f)

	newNumGlyphs := uint16(len(glyphs))

	newLocaData, newGlyfData, newHmtxData := buildGlyphTables(glyphs, locaFormat)

	var newMaxpData []byte
	if len(maxpData) >= ttfMaxpMinLen {
		// codehound-ignore: PERF-226
		newMaxpData = make([]byte, len(maxpData))
		copy(newMaxpData, maxpData)
		newMaxpData[ttfDWordSize] = byte(newNumGlyphs >> shift8)
		newMaxpData[ttfMaxpMinLen-1] = byte(newNumGlyphs)
	} else {
		newMaxpData = make([]byte, maxpFactoryLen)
		newMaxpData[0] = 0
		newMaxpData[1] = 0
		newMaxpData[ttfWordSize] = 1
		newMaxpData[ttfWordSize+1] = 0
		newMaxpData[ttfDWordSize] = byte(newNumGlyphs >> shift8)
		newMaxpData[ttfMaxpMinLen-1] = byte(newNumGlyphs)
		newMaxpData[ttfMaxpMinLen] = byte(newNumGlyphs >> shift8)
		newMaxpData[ttfMaxpMinLen+1] = byte(newNumGlyphs)
	}

	var newHeadData []byte
	if len(headData) >= ttfHeadMinLen {
		// codehound-ignore: PERF-226
		newHeadData = make([]byte, len(headData))
		copy(newHeadData, headData)
		newHeadData[headLocaFmtOff] = byte(locaFormat >> shift8)
		newHeadData[headLocaFmtOff+1] = byte(locaFormat)
		newHeadData[8] = 0
		newHeadData[9] = 0
		newHeadData[10] = 0
		newHeadData[11] = 0
	} else {
		return nil, errors.New("font: head table missing")
	}

	var newHheaData []byte
	if len(hheaData) >= ttfHheaMinLen {
		// codehound-ignore: PERF-226
		newHheaData = make([]byte, len(hheaData))
		copy(newHheaData, hheaData)
		newHheaData[ttfHheaMinLen-ttfWordSize] = byte(newNumGlyphs >> shift8)
		newHheaData[ttfHheaMinLen-1] = byte(newNumGlyphs)
	} else {
		return nil, errors.New("font: hhea table missing")
	}

	newCMapData := buildSubsetCMap(f, gidMap)

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
	if len(postData) >= ttfPostNameLen {
		// codehound-ignore: PERF-226
		newPostData := make([]byte, len(postData))
		copy(newPostData, postData)
		newPostData[ttfPostNameLen-ttfWordSize] = byte(newNumGlyphs >> shift8)
		newPostData[ttfPostNameLen-1] = byte(newNumGlyphs)
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
	header := buildTTFHeader(numTables)

	type tableInfo struct {
		tag    string
		data   []byte
		offset uint32
		length uint32
	}

	tableInfos := make([]tableInfo, 0, len(tables))
	off := uint32(ttfDirOffset) + uint32(numTables)*ttfEntrySize
	for _, t := range tables {
		length := uint32(len(t.data))
		tableInfos = append(tableInfos, tableInfo{
			tag: t.tag, data: t.data, offset: off, length: length,
		})
		off += pad4(length)
	}

	out := make([]byte, off)
	copy(out, header)

	dirBase := uint32(ttfDirOffset)
	for _, ti := range tableInfos {
		copy(out[dirBase:], ti.tag)
		out[dirBase+8] = byte(ti.offset >> shift24)
		out[dirBase+9] = byte(ti.offset >> shift16)
		out[dirBase+10] = byte(ti.offset >> shift8)
		out[dirBase+11] = byte(ti.offset)
		out[dirBase+12] = byte(ti.length >> shift24)
		out[dirBase+13] = byte(ti.length >> shift16)
		out[dirBase+14] = byte(ti.length >> shift8)
		out[dirBase+15] = byte(ti.length)
		dirBase += ttfEntrySize
	}

	for _, ti := range tableInfos {
		copy(out[ti.offset:], ti.data)
	}

	// Calculate and set checkSumAdjustment
	checksum := calcFileChecksum(out)
	adjustment := uint32(ttfOffsetMagic) - checksum
	headOff := findTableOffset(out, "head")
	if headOff != 0 {
		out[headOff+8] = byte(adjustment >> shift24)
		out[headOff+9] = byte(adjustment >> shift16)
		out[headOff+10] = byte(adjustment >> shift8)
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
	if len(data) < ttfDirOffset {
		return 0
	}
	numTables := binary.BigEndian.Uint16(data[ttfDWordSize:])
	for i := uint16(0); i < numTables; i++ {
		base := uint32(ttfDirOffset) + uint32(i)*ttfEntrySize
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

	glyphIDArray := make([]uint16, 0, len(sorted))
	for _, r := range sorted {
		glyphIDArray = append(glyphIDArray, gidMap[r])
	}

	headerLen := uint16(cmapF4BaseOffset)
	endCodesLen := uint16(segCount) * ttfWordSize
	startCodesLen := uint16(segCount) * ttfWordSize
	deltasLen := uint16(segCount) * ttfWordSize
	rangeOffsetsLen := uint16(segCount) * ttfWordSize
	glyphArrayLen := uint16(len(glyphIDArray)) * ttfWordSize

	totalLength := headerLen + endCodesLen + cmapF4ReservedPad + startCodesLen + deltasLen + rangeOffsetsLen + glyphArrayLen

	data := make([]byte, totalLength)

	data[0] = 0
	data[1] = 4
	data[ttfWordSize] = byte(totalLength >> shift8)
	data[ttfDWordSize-1] = byte(totalLength)
	data[4] = 0
	data[5] = 0
	segCountX2 := uint16(segCount) * ttfWordSize
	data[ttfMaxpMinLen] = byte(segCountX2 >> shift8)
	data[ttfMaxpMinLen+1] = byte(segCountX2)

	srEntrySelector := uint16(0)
	for 1<<(srEntrySelector+1) <= uint16(segCount) {
		srEntrySelector++
	}
	srSearchRangeBytes := uint16(1<<srEntrySelector) * ttfWordSize
	data[8] = byte(srSearchRangeBytes >> shift8)
	data[9] = byte(srSearchRangeBytes)
	data[10] = byte(srEntrySelector >> shift8)
	data[11] = byte(srEntrySelector)
	v := uint16(segCount)*ttfWordSize - srSearchRangeBytes
	data[12] = byte(v >> shift8)
	data[13] = byte(v)

	off := uint16(cmapF4BaseOffset)
	for i := 0; i < segCount; i++ {
		data[off] = byte(endCodes[i] >> shift8)
		data[off+1] = byte(endCodes[i])
		off += ttfWordSize
	}
	data[off] = 0
	data[off+1] = 0
	off += ttfWordSize
	for i := 0; i < segCount; i++ {
		data[off] = byte(startCodes[i] >> shift8)
		data[off+1] = byte(startCodes[i])
		off += ttfWordSize
	}
	for i := 0; i < segCount; i++ {
		data[off] = byte(uint16(idDeltas[i]) >> shift8)
		data[off+1] = byte(uint16(idDeltas[i]))
		off += ttfWordSize
	}

	for i := 0; i < segCount; i++ {
		data[off] = 0
		data[off+1] = 0
		off += ttfWordSize
	}

	for _, gid := range glyphIDArray {
		data[off] = byte(gid >> shift8)
		data[off+1] = byte(gid)
		off += ttfWordSize
	}

	return data
}

func buildEmptyFormat4CMap() []byte {
	segCount := uint16(1)
	segCountX2 := segCount * ttfWordSize
	totalLength := uint16(cmapF4BaseOffset + cmapF4ReservedPad + ttfWordSize + ttfWordSize + ttfWordSize + 0)
	data := make([]byte, totalLength)
	data[0] = 0
	data[1] = cmapFmt4
	data[ttfWordSize] = byte(totalLength >> shift8)
	data[ttfDWordSize-1] = byte(totalLength)
	data[ttfDWordSize] = 0
	data[ttfDWordSize+1] = 0
	data[ttfMaxpMinLen] = byte(segCountX2 >> shift8)
	data[ttfMaxpMinLen+1] = byte(segCountX2)
	data[8] = 0
	data[9] = ttfWordSize
	data[10] = 0
	data[11] = 0
	data[12] = 0
	data[13] = 0
	data[cmapF4BaseOffset] = 0xFF
	data[cmapF4BaseOffset+1] = 0xFF
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
