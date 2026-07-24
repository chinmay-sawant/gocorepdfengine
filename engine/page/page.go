package page

import (
	"fmt"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

type Page struct {
	MediaBox            [4]float64
	ContentsRef         doc.ObjectID
	FontResources       map[string]doc.ObjectID
	XObjectResources    map[string]doc.ObjectID
	ColorSpaceResources map[string]interface{}
	StructParents       *int
	Tabs                string
}

type Pages struct {
	Kids  []doc.ObjectID
	Count int
}

func NewPages() *Pages {
	return &Pages{}
}

func NewPage(width, height float64) *Page {
	return &Page{
		MediaBox: [4]float64{0, 0, width, height},
	}
}

func (p *Page) ToDict(fontMap, xobjMap map[string]doc.ObjectID, pagesRef doc.ObjectID) map[string]interface{} {
	dict := map[string]interface{}{
		"/Type":     "/Page",
		"/Parent":   fmt.Sprintf("%d 0 R", pagesRef),
		"/MediaBox": []interface{}{p.MediaBox[0], p.MediaBox[1], p.MediaBox[2], p.MediaBox[3]},
	}
	if p.ContentsRef != 0 {
		dict["/Contents"] = fmt.Sprintf("%d 0 R", p.ContentsRef)
	}
	res := make(map[string]interface{})
	hasRes := false
	if len(fontMap) > 0 {
		fd := make(map[string]interface{})
		for name, ref := range fontMap {
			fd["/"+name] = fmt.Sprintf("%d 0 R", ref)
		}
		res["/Font"] = fd
		hasRes = true
	}
	if len(xobjMap) > 0 {
		xd := make(map[string]interface{})
		for name, ref := range xobjMap {
			xd["/"+name] = fmt.Sprintf("%d 0 R", ref)
		}
		res["/XObject"] = xd
		hasRes = true
	}
	if p.ColorSpaceResources != nil {
		res["/ColorSpace"] = p.ColorSpaceResources
		hasRes = true
	}
	if hasRes {
		dict["/Resources"] = res
	}
	if p.StructParents != nil {
		dict["/StructParents"] = *p.StructParents
	}
	if p.Tabs != "" {
		dict["/Tabs"] = "/" + p.Tabs
	}
	return dict
}

func (p *Pages) ToDict() map[string]interface{} {
	kids := make([]interface{}, len(p.Kids))
	for i, kid := range p.Kids {
		kids[i] = fmt.Sprintf("%d 0 R", kid)
	}
	return map[string]interface{}{
		"/Type":  "/Pages",
		"/Kids":  kids,
		"/Count": p.Count,
	}
}
