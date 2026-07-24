// Package page provides PDF page and page-tree node types for the document
// structure. Pages hold content-stream references, font/XObject/color-space
// resource maps, and optional structural tagging metadata.
package page

import (
	"strconv"

	"github.com/chinmay/gocorepdfengine/engine/doc"
)

// Page represents a single PDF page node with its resources and structure
// tagging information.
type Page struct {
	MediaBox            [4]float64
	ContentsRef         doc.ObjectID
	FontResources       map[string]doc.ObjectID
	XObjectResources    map[string]doc.ObjectID
	ColorSpaceResources map[string]interface{}
	StructParents       *int
	Tabs                string
}

// Pages is the intermediate page-tree node that collects individual Pages.
type Pages struct {
	Kids  []doc.ObjectID
	Count int
}

// NewPages creates an empty Pages node.
func NewPages() *Pages {
	return &Pages{}
}

// NewPage creates a Page with the given media-box dimensions.
func NewPage(width, height float64) *Page {
	return &Page{
		MediaBox: [4]float64{0, 0, width, height},
	}
}

// ToDict converts the Page into a PDF dictionary (map[string]interface{}).
// fontMap and xobjMap are resolved to indirect object references by name.
func (p *Page) ToDict(fontMap, xobjMap map[string]doc.ObjectID, pagesRef doc.ObjectID) map[string]interface{} {
	dict := map[string]interface{}{
		"/Type":     "/Page",
		"/Parent":   strconv.Itoa(int(pagesRef)) + " 0 R",
		"/MediaBox": []interface{}{p.MediaBox[0], p.MediaBox[1], p.MediaBox[2], p.MediaBox[3]},
	}
	if p.ContentsRef != 0 {
		dict["/Contents"] = strconv.Itoa(int(p.ContentsRef)) + " 0 R"
	}
	res := make(map[string]interface{})
	hasRes := false
	if len(fontMap) > 0 {
		fd := make(map[string]interface{}, len(fontMap))
		for name, ref := range fontMap { // map iteration, fine
			fd["/"+name] = strconv.Itoa(int(ref)) + " 0 R" // unavoidable per-entry formatting
		}
		res["/Font"] = fd
		hasRes = true
	}
	if len(xobjMap) > 0 {
		xd := make(map[string]interface{}, len(xobjMap))
		for name, ref := range xobjMap { // map iteration, fine
			xd["/"+name] = strconv.Itoa(int(ref)) + " 0 R" // unavoidable per-entry formatting
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

// ToDict converts the Pages node into a PDF dictionary with Kids and Count.
func (p *Pages) ToDict() map[string]interface{} {
	kids := make([]interface{}, len(p.Kids))
	for i, kid := range p.Kids {
		kids[i] = strconv.Itoa(int(kid)) + " 0 R" // unavoidable per-entry formatting
	}
	return map[string]interface{}{
		"/Type":  "/Pages",
		"/Kids":  kids,
		"/Count": p.Count,
	}
}
