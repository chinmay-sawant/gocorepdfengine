// codehound-ignore-file: BP-1
package font

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"sync"
)

// codehound-ignore: PERF-110
var glyphBufPool = sync.Pool{
	New: func() any {
		return make([]byte, 0, glyphBufInitSize)
	},
}

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
		return errf("font: tableDir", err)
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
		return errf("font: subset generation error", err)
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

	bufPtr, _ := glyphBufPool.Get().(*[]byte)
	var buf []byte
	if bufPtr != nil {
		buf = *bufPtr
	}
	if cap(buf) < int(totalGlyphSize) {
		buf = make([]byte, totalGlyphSize)
	}
	glyphDataBuf := buf[:totalGlyphSize]
	defer glyphBufPool.Put(&buf)
	var dataOffset uint32
	var glyphs = make([]glyphEntry, 0, len(infos))
	for _, info := range infos {
		var data []byte
		if info.length > 0 {
			data = glyphDataBuf[dataOffset : dataOffset+info.length]
			copy(data, glyfTable[info.offset:info.offset+info.length])
			if isCompositeGlyph(data) {
				remapCompositeGIDs(data, gidMap)
			}
		}
		glyphs = append(glyphs, glyphEntry{
			gid: info.gid, data: data, length: info.length, width: info.width,
		})
		dataOffset += pad4(info.length)
	}

	return glyphs, gidMap
}

// Composite glyph flag bits (TrueType glyf table).
const (
	glyfArg1And2AreWords = 0x0001
	glyfWeHaveAScale     = 0x0008
	glyfMoreComponents   = 0x0020
	glyfWeHaveXYScale    = 0x0040
	glyfWeHave2By2       = 0x0080
)

func isCompositeGlyph(data []byte) bool {
	if len(data) < 10 {
		return false
	}
	return int16(binary.BigEndian.Uint16(data[0:2])) < 0
}

func compositeComponentGIDs(data []byte) []uint16 {
	if !isCompositeGlyph(data) {
		return nil
	}
	var gids []uint16
	off := 10
	for off+4 <= len(data) {
		flags := binary.BigEndian.Uint16(data[off:])
		gids = append(gids, binary.BigEndian.Uint16(data[off+2:]))
		off += 4
		if flags&glyfArg1And2AreWords != 0 {
			off += 4
		} else {
			off += 2
		}
		switch {
		case flags&glyfWeHaveAScale != 0:
			off += 2
		case flags&glyfWeHaveXYScale != 0:
			off += 4
		case flags&glyfWeHave2By2 != 0:
			off += 8
		}
		if flags&glyfMoreComponents == 0 {
			break
		}
	}
	return gids
}

func remapCompositeGIDs(data []byte, gidMap map[uint16]uint16) {
	if !isCompositeGlyph(data) {
		return
	}
	off := 10
	for off+4 <= len(data) {
		flags := binary.BigEndian.Uint16(data[off:])
		oldGID := binary.BigEndian.Uint16(data[off+2:])
		if newGID, ok := gidMap[oldGID]; ok {
			data[off+2] = byte(newGID >> shift8)
			data[off+3] = byte(newGID)
		} else {
			data[off+2] = 0
			data[off+3] = 0
		}
		off += 4
		if flags&glyfArg1And2AreWords != 0 {
			off += 4
		} else {
			off += 2
		}
		switch {
		case flags&glyfWeHaveAScale != 0:
			off += 2
		case flags&glyfWeHaveXYScale != 0:
			off += 4
		case flags&glyfWeHave2By2 != 0:
			off += 8
		}
		if flags&glyfMoreComponents == 0 {
			break
		}
	}
}

// expandCompositeDependencies adds component glyphs of composites so subset
// fonts keep referenced outlines and remapped component indices stay valid.
func expandCompositeDependencies(orig []byte, usedGIDs map[uint16]bool) {
	glyfTable, err := findTable(orig, "glyf")
	if err != nil || len(glyfTable) == 0 {
		return
	}
	locaTable, err := findTable(orig, "loca")
	if err != nil || len(locaTable) == 0 {
		return
	}
	headData, err := findTable(orig, "head")
	if err != nil || len(headData) < headLocaFmtOff+ttfWordSize {
		return
	}
	locaFormat := binary.BigEndian.Uint16(headData[headLocaFmtOff:])

	queue := make([]uint16, 0, len(usedGIDs))
	for gid := range usedGIDs {
		queue = append(queue, gid)
	}
	for len(queue) > 0 {
		gid := queue[0]
		queue = queue[1:]
		offset, length := readGlyphOffsetLength(gid, locaTable, locaFormat)
		if length == 0 || uint32(len(glyfTable)) < offset+length {
			continue
		}
		for _, comp := range compositeComponentGIDs(glyfTable[offset : offset+length]) {
			if !usedGIDs[comp] {
				usedGIDs[comp] = true
				queue = append(queue, comp)
			}
		}
	}
}

func buildGlyphTables(glyphs []glyphEntry, locaFormat uint16) ([]byte, []byte, []byte) {
	// loca has numGlyphs+1 entries: start offset of each glyph plus the end of the last.
	var newLocaData []byte
	if locaFormat == 0 {
		var glyphOffset uint32
		for _, ge := range glyphs {
			v := uint16(glyphOffset / ttfWordSize)
			newLocaData = append(newLocaData, byte(v>>shift8), byte(v))
			glyphOffset += pad4(ge.length)
		}
		v := uint16(glyphOffset / ttfWordSize)
		newLocaData = append(newLocaData, byte(v>>shift8), byte(v))
	} else {
		var glyphOffset uint32
		for _, ge := range glyphs {
			newLocaData = append(newLocaData,
				byte(glyphOffset>>shift24), byte(glyphOffset>>shift16),
				byte(glyphOffset>>shift8), byte(glyphOffset))
			glyphOffset += pad4(ge.length)
		}
		newLocaData = append(newLocaData,
			byte(glyphOffset>>shift24), byte(glyphOffset>>shift16),
			byte(glyphOffset>>shift8), byte(glyphOffset))
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
	subtable := buildFormat4CMap(f, gidMap)
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

	// sfntVersion must be 0x00010000 (bytes 00 01 00 00) for TrueType.
	header := make([]byte, ttfDirOffset)
	header[0] = 0
	header[1] = 1
	header[2] = 0
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

func tableChecksum(data []byte) uint32 {
	var sum uint32
	n := len(data)
	i := 0
	for ; i+3 < n; i += 4 {
		sum += binary.BigEndian.Uint32(data[i:])
	}
	if rem := n - i; rem > 0 {
		var last uint32
		for j := 0; j < rem; j++ {
			last |= uint32(data[i+j]) << (shift8 * (3 - j))
		}
		sum += last
	}
	return sum
}

// codehound-ignore: BP-1
func buildSubsetTTF(f *Font, orig []byte, _ []tableDirEntry, usedGIDs map[uint16]bool, _ map[string]bool) ([]byte, error) {
	// Pull in composite components before ordering GIDs.
	expandCompositeDependencies(orig, usedGIDs)

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
		// Copy before mutate — maxpData is a slice into shared RawData.
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
		// checkSumAdjustment filled after the full file is assembled.
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
	// Format 3.0 post: metrics only, no glyph names (valid for PDF embedding).
	postHeader := make([]byte, postMinLen)
	postHeader[0] = 0
	postHeader[1] = 3
	if origPost := findTableData("post"); len(origPost) >= postMinLen {
		copy(postHeader[4:], origPost[4:postMinLen])
	}
	addTable("post", postHeader)
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
		tag      string
		data     []byte
		offset   uint32
		length   uint32
		checksum uint32
	}

	tableInfos := make([]tableInfo, 0, len(tables))
	off := uint32(ttfDirOffset) + uint32(numTables)*ttfEntrySize
	for _, t := range tables {
		length := uint32(len(t.data))
		tableInfos = append(tableInfos, tableInfo{
			tag: t.tag, data: t.data, offset: off, length: length,
			checksum: tableChecksum(t.data),
		})
		off += pad4(length)
	}

	out := make([]byte, off)
	copy(out, header)

	dirBase := uint32(ttfDirOffset)
	for _, ti := range tableInfos {
		copy(out[dirBase:], ti.tag)
		out[dirBase+4] = byte(ti.checksum >> shift24)
		out[dirBase+5] = byte(ti.checksum >> shift16)
		out[dirBase+6] = byte(ti.checksum >> shift8)
		out[dirBase+7] = byte(ti.checksum)
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

// buildFormat4CMap builds a format-4 cmap that maps Unicode code points to
// *new* subset glyph IDs. gidMap is oldGID → newGID.
func buildFormat4CMap(f *Font, gidMap map[uint16]uint16) []byte {
	usedChars := f.UsedChars()

	type mapping struct {
		code uint16
		gid  uint16
	}
	maps := make([]mapping, 0, len(usedChars))
	for _, r := range usedChars {
		if r < 0 || r > 0xFFFF {
			continue
		}
		oldGID := uint16(0)
		if g, ok := f.Glyphs[r]; ok {
			oldGID = g.GID
		}
		newGID := gidMap[oldGID]
		// Skip .notdef for non-null control; keep explicit space/etc.
		if newGID == 0 && r != 0 {
			// Still emit mapping to .notdef so code is present if needed.
		}
		maps = append(maps, mapping{code: uint16(r), gid: newGID})
	}
	sort.Slice(maps, func(i, j int) bool { return maps[i].code < maps[j].code })

	if len(maps) == 0 {
		return buildEmptyFormat4CMap()
	}

	// Contiguous Unicode ranges; glyph IDs are stored in glyphIDArray via
	// non-zero idRangeOffset (idDelta alone is wrong when codes ≠ GIDs).
	type segRange struct {
		start, end int // indices into maps
	}
	var ranges []segRange
	rs := 0
	for i := 1; i < len(maps); i++ {
		if maps[i].code == maps[i-1].code+1 {
			continue
		}
		ranges = append(ranges, segRange{start: rs, end: i - 1})
		rs = i
	}
	ranges = append(ranges, segRange{start: rs, end: len(maps) - 1})

	segCount := len(ranges) + 1 // + sentinel 0xFFFF

	endCodes := make([]uint16, segCount)
	startCodes := make([]uint16, segCount)
	idDeltas := make([]int16, segCount)
	idRangeOffsets := make([]uint16, segCount)

	glyphIDArray := make([]uint16, 0, len(maps))
	for _, m := range maps {
		glyphIDArray = append(glyphIDArray, m.gid)
	}

	// idRangeOffset is byte offset from the offset field itself into glyphIDArray.
	// For segment i, offset = 2 * ( (segCount-i) + indexOfFirstGlyphInArray )
	// where indexOfFirstGlyphInArray is the running count of codes before this segment.
	codeIndex := 0
	for i, r := range ranges {
		startCodes[i] = maps[r.start].code
		endCodes[i] = maps[r.end].code
		idDeltas[i] = 0
		// bytes from this idRangeOffset entry to glyphIDArray[codeIndex]:
		// remaining idRangeOffset entries after this one: (segCount-1-i)
		// then glyphIDArray starts; skip codeIndex entries.
		// idRangeOffset unit is bytes; each entry is 2 bytes.
		idRangeOffsets[i] = uint16((segCount-i)+codeIndex) * ttfWordSize
		codeIndex += r.end - r.start + 1
	}
	endCodes[segCount-1] = 0xFFFF
	startCodes[segCount-1] = 0xFFFF
	idDeltas[segCount-1] = 1
	idRangeOffsets[segCount-1] = 0

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
		data[off] = byte(idRangeOffsets[i] >> shift8)
		data[off+1] = byte(idRangeOffsets[i])
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

func errf(msg string, err error) error {
	return errors.Join(errors.New(msg), err)
}

func errfs(format string, args ...any) error {
	// codehound-ignore: PERF-35
	return fmt.Errorf(format, args...)
}
