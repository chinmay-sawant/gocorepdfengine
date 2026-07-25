// codehound-ignore-file: BP-27

package structure

import "github.com/chinmay/gocorepdfengine/engine/doc"

// Manager coordinates the lifecycle of structure elements: allocation of
// marked-content identifiers, element creation, and final assembly of the
// structure tree dictionaries.
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

// NewManager creates a Manager. When enabled is false, all methods return
// zero values (producing no structure output).
func NewManager(enabled bool, allocID func() doc.ObjectID) *Manager {
	return &Manager{
		Enabled:    enabled,
		PageMCIDs:  make(map[int]int),
		ParentTree: make(map[int][]doc.ObjectID),
		AnnotTree:  make(map[int]doc.ObjectID),
		NextObjID:  allocID,
	}
}

// AllocMCID returns the next marked-content identifier for the given page.
func (m *Manager) AllocMCID(pageIndex int) int {
	if !m.Enabled {
		return 0
	}
	mcid := m.PageMCIDs[pageIndex]
	m.PageMCIDs[pageIndex] = mcid + 1
	return mcid
}

// AddElement assigns a new object ID to the element and appends it to the
// element list. Returns the assigned ID.
func (m *Manager) AddElement(elem *StructElem) doc.ObjectID {
	if !m.Enabled {
		return 0
	}
	id := m.NextObjID()
	elem.ObjectID = id
	m.Elements = append(m.Elements, elem)
	return id
}

// SetDocumentRoot creates the root /Document element and registers it.
func (m *Manager) SetDocumentRoot() {
	if !m.Enabled {
		return
	}
	elem := &StructElem{
		Type:   TypeDocument,
		Parent: 0,
		MCID:   -1,
	}
	m.AddElement(elem)
	m.Root = elem
}

// Build assembles and returns all structure tree dictionaries (namespace,
// root, parent tree) and their object references. Returns nil values when
// the manager is disabled.
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
