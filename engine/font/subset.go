package font

import (
	"encoding/binary"
	"fmt"
	"sort"
)

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

func (f *Font) AddChars(chars []rune) {
	for _, r := range chars {
		f.AddChar(r)
	}
}

func (f *Font) UsedChars() []rune {
	return f.UsedRunes()
}

func (f *Font) GenerateSubset() error {
	if len(f.RawData) < 12 {
		return fmt.Errorf("font: no raw data to subset")
	}

	usedGIDs := buildUsedGIDSet(f)
	usedGIDs[0] = true

	if len(usedGIDs) <= 1 {
		for _, g := range f.Glyphs {
			usedGIDs[g.GID] = true
		}
	}

	entries, err := tableDir(f.RawData)
	if err != nil {
		return err
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
		return fmt.Errorf("font: generated subset is invalid (too short)")
	}

	f.SubsetData = subset
	return nil
}

func buildUsedGIDSet(f *Font) map[uint16]bool {
	used := make(map[uint16]bool)
	for _, g := range f.Glyphs {
		used[g.GID] = true
	}
	return used
}

func pad4(n uint32) uint32 {
	return (n + 3) & ^uint32(3)
}

func buildSubsetTTF(f *Font, orig []byte, entries []tableDirEntry, usedGIDs map[uint16]bool, required map[string]bool) ([]byte, error) {
	gidList := make([]uint16, 0, len(usedGIDs))
	for gid := range usedGIDs {
		gidList = append(gidList, gid)
	}
	sort.Slice(gidList, func(i, j int) bool { return gidList[i] < gidList[j] })

	maxpData, _ := findTable(orig, "maxp")
	oldNumGlyphs := uint16(0)
	if maxpData != nil && len(maxpData) >= 6 {
		oldNumGlyphs = binary.BigEndian.Uint16(maxpData[4:])
	}

	headData, _ := findTable(orig, "head")
	locaFormat := uint16(0)
	if headData != nil && len(headData) >= 52 {
		locaFormat = binary.BigEndian.Uint16(headData[50:])
	}

	hheaData, _ := findTable(orig, "hhea")
	numHMetrics := oldNumGlyphs
	if hheaData != nil && len(hheaData) >= 36 {
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

	gidMap := make(map[uint16]uint16)
	for newID, oldGID := range gidList {
		gidMap[oldGID] = uint16(newID)
	}
	f.SubGIDMap = gidMap

	for _, oldGID := range gidList {
		var data []byte
		var length uint32

		if glyfTable != nil && locaTable != nil {
			var glyphOffset, nextOffset uint32
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
				data = make([]byte, length)
				copy(data, glyfTable[glyphOffset:glyphOffset+length])
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

		glyphs = append(glyphs, glyphEntry{
			gid: oldGID, data: data, length: length, width: width,
		})
	}

	newNumGlyphs := uint16(len(glyphs))

	var newLocaData []byte
	if locaFormat == 0 {
		var glyphOffset uint32 = 0
		for i, ge := range glyphs {
			b := make([]byte, 4)
			binary.BigEndian.PutUint16(b[0:], uint16(glyphOffset/2))
			glyphOffset += pad4(ge.length)
			binary.BigEndian.PutUint16(b[2:], uint16(glyphOffset/2))
			newLocaData = append(newLocaData, b...)
			_ = i
		}
	} else {
		var glyphOffset uint32 = 0
		for i, ge := range glyphs {
			b := make([]byte, 8)
			binary.BigEndian.PutUint32(b[0:], glyphOffset)
			glyphOffset += pad4(ge.length)
			binary.BigEndian.PutUint32(b[4:], glyphOffset)
			newLocaData = append(newLocaData, b...)
			_ = i
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

	var newHmtxData []byte
	for _, ge := range glyphs {
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, ge.width)
		newHmtxData = append(newHmtxData, b...)
	}
	for range glyphs {
		b := make([]byte, 2)
		newHmtxData = append(newHmtxData, b...)
	}

	var newMaxpData []byte
	if maxpData != nil && len(maxpData) >= 6 {
		newMaxpData = make([]byte, len(maxpData))
		copy(newMaxpData, maxpData)
		binary.BigEndian.PutUint16(newMaxpData[4:], newNumGlyphs)
	} else {
		newMaxpData = make([]byte, 32)
		binary.BigEndian.PutUint32(newMaxpData, 0x00010000)
		binary.BigEndian.PutUint16(newMaxpData[4:], newNumGlyphs)
		binary.BigEndian.PutUint16(newMaxpData[6:], newNumGlyphs)
	}

	var newHeadData []byte
	if headData != nil && len(headData) >= 54 {
		newHeadData = make([]byte, len(headData))
		copy(newHeadData, headData)
		binary.BigEndian.PutUint16(newHeadData[50:], locaFormat)
		binary.BigEndian.PutUint32(newHeadData[8:], 0)
	} else {
		return nil, fmt.Errorf("font: head table missing")
	}

	var newHheaData []byte
	if hheaData != nil && len(hheaData) >= 36 {
		newHheaData = make([]byte, len(hheaData))
		copy(newHheaData, hheaData)
		binary.BigEndian.PutUint16(newHheaData[34:], newNumGlyphs)
	} else {
		return nil, fmt.Errorf("font: hhea table missing")
	}

	var newCMapData []byte
	{
		usedChars := f.UsedChars()
		subtable := buildFormat4CMap(usedChars, gidMap)
		// Wrap in cmap table header (version + numTables + encoding record)
		headerLen := uint32(4 + 8)
		cmap := make([]byte, headerLen+uint32(len(subtable)))
		binary.BigEndian.PutUint16(cmap, 0)      // version
		binary.BigEndian.PutUint16(cmap[2:], 1)   // numTables
		binary.BigEndian.PutUint16(cmap[4:], 3)   // platformID = Microsoft
		binary.BigEndian.PutUint16(cmap[6:], 1)   // encodingID = Unicode BMP
		binary.BigEndian.PutUint32(cmap[8:], headerLen) // offset to subtable
		copy(cmap[headerLen:], subtable)
		newCMapData = cmap
	}

	copyTable := func(tag string) []byte {
		d, err := findTable(orig, tag)
		if err != nil {
			return nil
		}
		out := make([]byte, len(d))
		copy(out, d)
		return out
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
	addTable("OS/2", copyTable("OS/2"))
	addTable("name", copyTable("name"))
	addTable("cmap", newCMapData)
	postData := copyTable("post")
	if len(postData) >= 34 {
		newPostData := make([]byte, len(postData))
		copy(newPostData, postData)
		binary.BigEndian.PutUint16(newPostData[32:], newNumGlyphs)
		postData = newPostData
	}
	addTable("post", postData)
	addTable("loca", newLocaData)
	addTable("glyf", newGlyfData)
	addTable("hmtx", newHmtxData)

	cvt := copyTable("cvt ")
	if cvt == nil {
		cvt = copyTable("cvt")
	}
	addTable("prep", copyTable("prep"))
	addTable("fpgm", copyTable("fpgm"))
	addTable("cvt ", cvt)

	numTables := uint16(len(tables))

	entrySelector := uint16(0)
	for 1<<(entrySelector+1) <= numTables {
		entrySelector++
	}
	searchRange := uint16(1 << entrySelector) * 16
	rangeShift := numTables*16 - searchRange

	header := make([]byte, 12)
	binary.BigEndian.PutUint32(header, 0x00010000)
	binary.BigEndian.PutUint16(header[4:], numTables)
	binary.BigEndian.PutUint16(header[6:], searchRange)
	binary.BigEndian.PutUint16(header[8:], entrySelector)
	binary.BigEndian.PutUint16(header[10:], rangeShift)

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
		copy(out[dirBase:], []byte(ti.tag))
		binary.BigEndian.PutUint32(out[dirBase+8:], ti.offset)
		binary.BigEndian.PutUint32(out[dirBase+12:], ti.length)
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
		binary.BigEndian.PutUint32(out[headOff+8:], adjustment)
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

	binary.BigEndian.PutUint16(data, 4)
	binary.BigEndian.PutUint16(data[2:], totalLength)
	binary.BigEndian.PutUint16(data[4:], 0)
	segCountX2 := uint16(segCount) * 2
	binary.BigEndian.PutUint16(data[6:], segCountX2)

	srEntrySelector := uint16(0)
	for 1<<(srEntrySelector+1) <= uint16(segCount) {
		srEntrySelector++
	}
	srSearchRangeBytes := uint16(1 << srEntrySelector) * 2
	binary.BigEndian.PutUint16(data[8:], srSearchRangeBytes)
	binary.BigEndian.PutUint16(data[10:], srEntrySelector)
	binary.BigEndian.PutUint16(data[12:], uint16(segCount)*2-srSearchRangeBytes)

	off := uint16(14)
	for i := 0; i < segCount; i++ {
		binary.BigEndian.PutUint16(data[off:], endCodes[i])
		off += 2
	}
	binary.BigEndian.PutUint16(data[off:], 0)
	off += 2
	for i := 0; i < segCount; i++ {
		binary.BigEndian.PutUint16(data[off:], startCodes[i])
		off += 2
	}
	for i := 0; i < segCount; i++ {
		binary.BigEndian.PutUint16(data[off:], uint16(idDeltas[i]))
		off += 2
	}

	for i := 0; i < segCount; i++ {
		binary.BigEndian.PutUint16(data[off:], 0)
		off += 2
	}

	for _, gid := range glyphIDArray {
		binary.BigEndian.PutUint16(data[off:], gid)
		off += 2
	}

	return data
}

func buildEmptyFormat4CMap() []byte {
	segCount := uint16(1)
	segCountX2 := segCount * 2
	totalLength := uint16(14 + 2 + 2 + 2 + 2 + 0)
	data := make([]byte, totalLength)
	binary.BigEndian.PutUint16(data, 4)
	binary.BigEndian.PutUint16(data[2:], totalLength)
	binary.BigEndian.PutUint16(data[4:], 0)
	binary.BigEndian.PutUint16(data[6:], segCountX2)
	binary.BigEndian.PutUint16(data[8:], 2)
	binary.BigEndian.PutUint16(data[10:], 0)
	binary.BigEndian.PutUint16(data[12:], 0)
	binary.BigEndian.PutUint16(data[14:], 0xFFFF)
	binary.BigEndian.PutUint16(data[16:], 0)
	binary.BigEndian.PutUint16(data[18:], 0xFFFF)
	binary.BigEndian.PutUint16(data[20:], 1)
	binary.BigEndian.PutUint16(data[22:], 0)
	return data
}

