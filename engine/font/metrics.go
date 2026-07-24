package font

import (
	"encoding/binary"
	"fmt"
)

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

	cmap := make([]byte, 0, 1024)

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

	appendStr(fmt.Sprintf("%d beginbfrange\n", len(ranges)))

	for _, r := range ranges {
		startHex := fmt.Sprintf("%04X", r.startCID)
		endHex := fmt.Sprintf("%04X", r.endCID)
		uniStr := fmt.Sprintf("%04X", r.startCID)
		appendStr(fmt.Sprintf("<%s> <%s> <%s>\n", startHex, endHex, uniStr))
	}

	appendStr("endbfrange\n")
	appendStr("endcmap\n")
	appendStr("CMapName currentdict /CMap defineresource pop\n")
	appendStr("end\n")
	appendStr("end\n")

	return cmap
}

func (f *Font) BuildCIDToGIDMap() []byte {
	data := make([]byte, 65536*2)
	for r := range f.Glyphs {
		cid := uint32(r)
		if cid >= 65536 {
			continue
		}
		if f.SubGIDMap != nil {
			if newGID, ok := f.SubGIDMap[f.cmap[r]]; ok {
				binary.BigEndian.PutUint16(data[cid*2:], newGID)
			}
		} else {
			binary.BigEndian.PutUint16(data[cid*2:], f.cmap[r])
		}
	}
	return data
}
