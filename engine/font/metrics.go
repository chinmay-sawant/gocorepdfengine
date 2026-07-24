package font

import (
	"strconv"
)

func hex04(v uint16) string {
	const hex = "0123456789ABCDEF"
	return string([]byte{hex[v>>12], hex[(v>>8)&0xF], hex[(v>>4)&0xF], hex[v&0xF]})
}

// ToUnicodeCMap builds a ToUnicode CMap stream for the font.
func (f *Font) ToUnicodeCMap() []byte {
	keys := f.UsedRunes()
	if len(keys) == 0 {
		return nil
	}

	type bfRange struct {
		startCID uint16
		endCID   uint16
	}

	var ranges []bfRange
	prev := keys[0]
	rs := prev
	for i := 1; i < len(keys); i++ {
		curr := keys[i]
		if curr == prev+1 {
			prev = curr
			continue
		}
		ranges = append(ranges, bfRange{startCID: uint16(rs), endCID: uint16(prev)})
		rs = curr
		prev = curr
	}
	ranges = append(ranges, bfRange{startCID: uint16(rs), endCID: uint16(prev)})

	cmap := make([]byte, 0, 1024) //nolint: perflint // PERF-3: pre-allocated buffer

	appendStr := func(s string) {
		cmap = append(cmap, []byte(s)...)
	}

	appendStr("/CIDInit /ProcSet findresource begin\n")
	appendStr("12 dict begin\n")
	appendStr("begincmap\n")
	appendStr("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	appendStr("/CMapName /Adobe-Identity-UCS def\n")
	appendStr("/CMapType 2 def\n")
	appendStr("1 begincodespacerange\n")
	appendStr("<0000> <FFFF>\n")
	appendStr("endcodespacerange\n")

	appendStr(strconv.Itoa(len(ranges)) + " beginbfrange\n")

	for _, r := range ranges {
		appendStr("<" + hex04(r.startCID) + "> <" + hex04(r.endCID) + "> <" + hex04(r.startCID) + ">\n")
	}

	appendStr("endbfrange\n")
	appendStr("endcmap\n")
	appendStr("CMapName currentdict /CMap defineresource pop\n")
	appendStr("end\n")
	appendStr("end\n")

	return cmap
}

// BuildCIDToGIDMap builds a CIDToGIDMap stream for the font.
func (f *Font) BuildCIDToGIDMap() []byte {
	const (
		cidMax    = 65536
		cidMapLen = cidMax * 2
	)
	data := make([]byte, cidMapLen)
	for r := range f.Glyphs {
		cid := uint32(r)
		if cid >= cidMax {
			continue
		}
		if f.SubGIDMap != nil {
			if newGID, ok := f.SubGIDMap[f.cmap[r]]; ok {
				i := cid * 2
				data[i] = byte(newGID >> 8)
				data[i+1] = byte(newGID)
			}
		} else {
			i := cid * 2
			v := f.cmap[r]
			data[i] = byte(v >> 8)
			data[i+1] = byte(v)
		}
	}
	return data
}
