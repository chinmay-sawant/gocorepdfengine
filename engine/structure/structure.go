package structure

import (
	"fmt"
	"sort"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

type StructType string

const (
	SDocument StructType = "/Document"
	SPart     StructType = "/Part"
	SSect     StructType = "/Sect"
	SDiv      StructType = "/Div"
	SH1       StructType = "/H1"
	SH2       StructType = "/H2"
	SP        StructType = "/P"
	STable    StructType = "/Table"
	STR       StructType = "/TR"
	STH       StructType = "/TH"
	STD       StructType = "/TD"
	SFigure   StructType = "/Figure"
	SLink     StructType = "/Link"
	SCaption  StructType = "/Caption"
	SL        StructType = "/L"
	SLI       StructType = "/LI"
	SLbl      StructType = "/Lbl"
	SLBody    StructType = "/LBody"
	SForm     StructType = "/Form"
)

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

type StructElemKid struct {
	IsMCID bool
	Ref    doc.ObjectID
	MCID   int
	OBJR   *OBJR
}

type OBJR struct {
	ObjRef  doc.ObjectID
	PageRef doc.ObjectID
}

func Namespace() map[string]interface{} {
	return map[string]interface{}{
		"/Type": "/Namespace",
		"/NS":   "(http://iso.org/pdf2/ssn)",
	}
}

func NamespaceRef(id doc.ObjectID) string {
	return fmt.Sprintf("%d 0 R", id)
}

func StructTreeRootDict(kidsRef, parentTreeRef, nsRef doc.ObjectID) map[string]interface{} {
	return map[string]interface{}{
		"/Type":       "/StructTreeRoot",
		"/K":          fmt.Sprintf("%d 0 R", kidsRef),
		"/ParentTree": fmt.Sprintf("%d 0 R", parentTreeRef),
		"/Namespaces": []interface{}{fmt.Sprintf("%d 0 R", nsRef)},
	}
}

func StructElemDict(se *StructElem) map[string]interface{} {
	d := map[string]interface{}{
		"/Type": "/StructElem",
		"/S":    string(se.Type),
	}

	if se.Parent != 0 {
		d["/P"] = fmt.Sprintf("%d 0 R", se.Parent)
	}
	if se.PageRef != 0 {
		d["/Pg"] = fmt.Sprintf("%d 0 R", se.PageRef)
	}
	if se.Title != "" {
		d["/T"] = fmt.Sprintf("(%s)", se.Title)
	}
	if se.Alt != "" {
		d["/Alt"] = fmt.Sprintf("(%s)", se.Alt)
	}
	if se.Lang != "" {
		d["/Lang"] = fmt.Sprintf("(%s)", se.Lang)
	}
	if se.NamespaceRef != 0 {
		d["/NS"] = fmt.Sprintf("%d 0 R", se.NamespaceRef)
	}

	if len(se.Kids) > 0 {
		kArray := make([]interface{}, 0, len(se.Kids))
		for _, kid := range se.Kids {
			switch {
			case kid.OBJR != nil:
				kArray = append(kArray, map[string]interface{}{
					"/Type": "/OBJR",
					"/Obj":  fmt.Sprintf("%d 0 R", kid.OBJR.ObjRef),
					"/Pg":   fmt.Sprintf("%d 0 R", kid.OBJR.PageRef),
				})
			case kid.IsMCID:
				kArray = append(kArray, kid.MCID)
			default:
				kArray = append(kArray, fmt.Sprintf("%d 0 R", kid.Ref))
			}
		}
		d["/K"] = kArray
	} else if se.MCID >= 0 {
		d["/K"] = []interface{}{se.MCID}
	}

	return d
}

func ParentTreeDict(nums map[int][]doc.ObjectID, annots map[int]doc.ObjectID) map[string]interface{} {
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

	numPairs := make([]interface{}, 0, len(keys)*2)
	for _, k := range keys {
		if refs, ok := nums[k]; ok {
			refList := make([]interface{}, 0, len(refs))
			for _, ref := range refs {
				refList = append(refList, fmt.Sprintf("%d 0 R", ref))
			}
			numPairs = append(numPairs, k, refList)
		} else if ref, ok := annots[k]; ok {
			numPairs = append(numPairs, k, fmt.Sprintf("%d 0 R", ref))
		}
	}

	return map[string]interface{}{
		"/Nums": numPairs,
	}
}

func StructParentsValue(pageIndex int) int {
	return pageIndex
}
