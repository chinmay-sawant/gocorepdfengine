package structure

import (
	"fmt"
	"sort"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

type StructType string

const (
	S_Document StructType = "Document"
	S_Part     StructType = "Part"
	S_Sect     StructType = "Sect"
	S_Div      StructType = "Div"
	S_H1       StructType = "H1"
	S_H2       StructType = "H2"
	S_P        StructType = "P"
	S_Table    StructType = "Table"
	S_TR       StructType = "TR"
	S_TH       StructType = "TH"
	S_TD       StructType = "TD"
	S_Figure   StructType = "Figure"
	S_Link     StructType = "Link"
	S_Caption  StructType = "Caption"
	S_L        StructType = "L"
	S_LI       StructType = "LI"
	S_Lbl      StructType = "Lbl"
	S_LBody    StructType = "LBody"
	S_Form     StructType = "Form"
)

type StructElem struct {
	Type     StructType
	Title    string
	Alt      string
	Lang     string
	PageRef  doc.ObjectID
	Parent   doc.ObjectID
	Kids     []StructElemKid
	MCID     int
	ObjectID doc.ObjectID
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

	if len(se.Kids) > 0 {
		kArray := make([]interface{}, 0, len(se.Kids))
		for _, kid := range se.Kids {
			if kid.OBJR != nil {
				kArray = append(kArray, map[string]interface{}{
					"/Type": "/OBJR",
					"/Obj":  fmt.Sprintf("%d 0 R", kid.OBJR.ObjRef),
					"/Pg":   fmt.Sprintf("%d 0 R", kid.OBJR.PageRef),
				})
			} else if kid.IsMCID {
				kArray = append(kArray, kid.MCID)
			} else {
				kArray = append(kArray, fmt.Sprintf("%d 0 R", kid.Ref))
			}
		}
		d["/K"] = kArray
	} else if se.MCID >= 0 {
		d["/K"] = se.MCID
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
