package font

import "fmt"

func (f *Font) ToUnicodeCMap() []byte {
	keys := f.UsedRunes()
	if len(keys) == 0 {
		return nil
	}

	type bfRange struct {
		startCID uint16
		endCID   uint16
		unicode  rune
	}

	var ranges []bfRange
	cid := uint16(1)

	for i, r := range keys {
		if i == 0 {
			ranges = append(ranges, bfRange{startCID: cid, endCID: cid, unicode: r})
			cid++
			continue
		}

		prev := keys[i-1]

		if r == prev+1 {
			ranges[len(ranges)-1].endCID = cid
		} else {
			ranges = append(ranges, bfRange{startCID: cid, endCID: cid, unicode: r})
		}
		cid++
	}

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

	appendStr(fmt.Sprintf("%d begindbfrange\n", len(ranges)))

	for _, r := range ranges {
		startHex := fmt.Sprintf("%04X", r.startCID)
		endHex := fmt.Sprintf("%04X", r.endCID)
		uniStr := fmt.Sprintf("%04X", r.unicode)
		appendStr(fmt.Sprintf("<%s> <%s> <%s>\n", startHex, endHex, uniStr))
	}

	appendStr("endbfrange\n")
	appendStr("endcmap\n")
	appendStr("CMapName currentdict /CMap defineresource pop\n")
	appendStr("end\n")
	appendStr("end\n")

	return cmap
}
