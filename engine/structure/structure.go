// codehound-ignore-file: BP-27

// Package structure builds PDF tagged-structure elements (StructTreeRoot,
// StructElem, ParentTree) required for PDF/UA-2 accessibility compliance.
package structure

import (
	"sort"
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

const (
	decimalBase    = 10
	pairMultiplier = 2
)

// StructType is a PDF structure type name (e.g. /Document, /Sect, /P).
type StructType string

const (
	TypeDocument StructType = "/Document"
	TypePart     StructType = "/Part"
	TypeSect     StructType = "/Sect"
	TypeDiv      StructType = "/Div"
	TypeH1       StructType = "/H1"
	TypeH2       StructType = "/H2"
	TypeP        StructType = "/P"
	TypeTable    StructType = "/Table"
	TypeTR       StructType = "/TR"
	TypeTH       StructType = "/TH"
	TypeTD       StructType = "/TD"
	TypeFigure   StructType = "/Figure"
	TypeLink     StructType = "/Link"
	TypeCaption  StructType = "/Caption"
	TypeL        StructType = "/L"
	TypeLI       StructType = "/LI"
	TypeLbl      StructType = "/Lbl"
	TypeLBody    StructType = "/LBody"
	TypeForm     StructType = "/Form"
)

// StructElem is a single node in the PDF tagged-structure tree.
type StructElem struct {
	Type         StructType
	Title        string
	Alt          string
	Lang         string
	PageRef      doc.ObjectID
	Parent       doc.ObjectID
	Kids         []StructElemKid
	MCID         int
	ObjectID     doc.ObjectID
	NamespaceRef doc.ObjectID
}

// StructElemKid is a child entry in a StructElem's /K array. It can be a
// marked-content reference (MCID), an indirect object reference, or an OBJR.
type StructElemKid struct {
	IsMCID bool
	Ref    doc.ObjectID
	MCID   int
	OBJR   *OBJR
}

// OBJR is an indirect object reference within a structure element's kid list.
type OBJR struct {
	ObjRef  doc.ObjectID
	PageRef doc.ObjectID
}

// Namespace returns a PDF Namespace dictionary for the ISO standard structure
// namespace.
func Namespace() map[string]interface{} {
	return map[string]interface{}{
		"/Type": "/Namespace",
		"/NS":   "(http://iso.org/pdf2/ssn)",
	}
}

// NamespaceRef formats an object ID as a PDF indirect reference string for use
// in the Namespaces array.
func NamespaceRef(id doc.ObjectID) string {
	var refBuf []byte
	refBuf = strconv.AppendInt(refBuf[:0], int64(id), decimalBase)
	return string(refBuf) + " 0 R"
}

// StructTreeRootDict returns the StructTreeRoot dictionary with the given
// references for its /K (kids), /ParentTree, and /Namespaces entries.
func StructTreeRootDict(kidsRef, parentTreeRef, nsRef doc.ObjectID) map[string]interface{} {
	var refBuf []byte
	refBuf = strconv.AppendInt(refBuf[:0], int64(kidsRef), decimalBase)
	k := string(refBuf) + " 0 R"
	refBuf = strconv.AppendInt(refBuf[:0], int64(parentTreeRef), decimalBase)
	pt := string(refBuf) + " 0 R"
	refBuf = strconv.AppendInt(refBuf[:0], int64(nsRef), decimalBase)
	ns := string(refBuf) + " 0 R"
	return map[string]interface{}{
		"/Type":       "/StructTreeRoot",
		"/K":          k,
		"/ParentTree": pt,
		"/Namespaces": []interface{}{ns},
	}
}

// StructElemDict converts a StructElem into its PDF dictionary representation.
func StructElemDict(se *StructElem) map[string]interface{} {
	var refBuf []byte
	d := make(map[string]interface{})
	d["/Type"] = "/StructElem"
	d["/S"] = string(se.Type)

	if se.Parent != 0 {
		refBuf = strconv.AppendInt(refBuf[:0], int64(se.Parent), decimalBase)
		d["/P"] = string(refBuf) + " 0 R"
	}
	if se.PageRef != 0 {
		refBuf = strconv.AppendInt(refBuf[:0], int64(se.PageRef), decimalBase)
		d["/Pg"] = string(refBuf) + " 0 R"
	}
	if se.Title != "" {
		d["/T"] = "(" + se.Title + ")"
	}
	if se.Alt != "" {
		d["/Alt"] = "(" + se.Alt + ")"
	}
	if se.Lang != "" {
		d["/Lang"] = "(" + se.Lang + ")"
	}
	if se.NamespaceRef != 0 {
		refBuf = strconv.AppendInt(refBuf[:0], int64(se.NamespaceRef), decimalBase)
		d["/NS"] = string(refBuf) + " 0 R"
	}

	if len(se.Kids) > 0 {
		kArray := make([]interface{}, 0, len(se.Kids))
		for _, kid := range se.Kids {
			switch {
			case kid.OBJR != nil:
				refBuf = strconv.AppendInt(refBuf[:0], int64(kid.OBJR.ObjRef), decimalBase)
				objStr := string(refBuf) + " 0 R"
				refBuf = strconv.AppendInt(refBuf[:0], int64(kid.OBJR.PageRef), decimalBase)
				pgStr := string(refBuf) + " 0 R"
				kArray = append(kArray, map[string]interface{}{
					"/Type": "/OBJR",
					"/Obj":  objStr,
					"/Pg":   pgStr,
				})
			case kid.IsMCID:
				kArray = append(kArray, kid.MCID)
			default:
				refBuf = strconv.AppendInt(refBuf[:0], int64(kid.Ref), decimalBase)
				kArray = append(kArray, string(refBuf)+" 0 R")
			}
		}
		d["/K"] = kArray
	} else if se.MCID >= 0 {
		d["/K"] = []interface{}{se.MCID}
	}

	return d
}

// ParentTreeDict builds the /ParentTree number-tree dictionary mapping page
// structure element IDs to their parent struct elements.
func ParentTreeDict(nums map[int][]doc.ObjectID, annots map[int]doc.ObjectID) map[string]interface{} {
	var refBuf []byte
	keys := make([]int, 0, len(nums)+len(annots))
	for k := range nums {
		keys = append(keys, k)
	}
	for k := range annots {
		if _, ok := nums[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Ints(keys)

	numPairs := make([]interface{}, 0, len(keys)*pairMultiplier)
	for _, k := range keys {
		if refs, ok := nums[k]; ok {
			refList := make([]interface{}, 0, len(refs))
			for _, ref := range refs {
				refBuf = strconv.AppendInt(refBuf[:0], int64(ref), decimalBase)
				refList = append(refList, string(refBuf)+" 0 R")
			}
			numPairs = append(numPairs, k, refList)
		} else if ref, ok := annots[k]; ok {
			refBuf = strconv.AppendInt(refBuf[:0], int64(ref), decimalBase)
			numPairs = append(numPairs, k, string(refBuf)+" 0 R")
		}
	}

	return map[string]interface{}{
		"/Nums": numPairs,
	}
}

// StructParentsValue returns the page index for use as /StructParents.
func StructParentsValue(pageIndex int) int {
	return pageIndex
}
