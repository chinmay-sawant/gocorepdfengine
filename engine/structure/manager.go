package structure

import "github.com/chinmay/gocorepdfengine/engine/doc"

type Manager struct {
	Enabled     bool
	Elements    []*StructElem
	MCIDCounter int
	PageMCIDs   map[int]int
	ParentTree  map[int][]doc.ObjectID
	AnnotTree   map[int]doc.ObjectID
	NextObjID   func() doc.ObjectID
	Root        *StructElem
}

func NewManager(enabled bool, allocID func() doc.ObjectID) *Manager {
	return &Manager{
		Enabled:    enabled,
		PageMCIDs:  make(map[int]int),
		ParentTree: make(map[int][]doc.ObjectID),
		AnnotTree:  make(map[int]doc.ObjectID),
		NextObjID:  allocID,
	}
}

func (m *Manager) AllocMCID(pageIndex int) int {
	if !m.Enabled {
		return 0
	}
	mcid := m.PageMCIDs[pageIndex]
	m.PageMCIDs[pageIndex] = mcid + 1
	return mcid
}

func (m *Manager) AddElement(elem *StructElem) doc.ObjectID {
	if !m.Enabled {
		return 0
	}
	id := m.NextObjID()
	elem.ObjectID = id
	m.Elements = append(m.Elements, elem)
	return id
}

func (m *Manager) SetDocumentRoot() {
	if !m.Enabled {
		return
	}
	elem := &StructElem{
		Type:   SDocument,
		Parent: 0,
		MCID:   -1,
	}
	m.AddElement(elem)
	m.Root = elem
}

func (m *Manager) Build() (map[string]interface{}, map[string]interface{}, map[string]interface{}, doc.ObjectID, doc.ObjectID, doc.ObjectID, []*StructElem) {
	if !m.Enabled {
		return nil, nil, nil, 0, 0, 0, nil
	}

	nsRef := m.NextObjID()
	namespaceDict := Namespace()

	ptRef := m.NextObjID()
	parentTreeDict := ParentTreeDict(m.ParentTree, m.AnnotTree)

	rootRef := m.Root.ObjectID
	rootDict := StructTreeRootDict(rootRef, ptRef, nsRef)

	allElems := m.Elements
	return namespaceDict, rootDict, parentTreeDict, nsRef, rootRef, ptRef, allElems
}
