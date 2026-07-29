package doc

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"time"

	"github.com/chinmay/gocorepdfengine/engine/write"
)

type ObjectID uint32

type Mode uint32

const (
	ModePDF20      Mode = 1 << 0
	ModePDFA4      Mode = 1 << 1
	ModePDFUA2     Mode = 1 << 2
	ModeEmbedFonts Mode = 1 << 3
)

type Object struct {
	ID   ObjectID
	Gen  uint16
	Data interface{}
}

type Document struct {
	Objects      []*Object
	nextID       ObjectID
	Mode         Mode
	catalogRef   ObjectID
	pagesRootRef ObjectID
	TrailerInfo  map[string]interface{}
}

func NewDocument() *Document {
	now := write.StringLit(write.DateString(time.Now()))
	return &Document{
		nextID: 1,
		Mode:   ModePDF20,
		TrailerInfo: map[string]interface{}{
			"/ModDate":      now,
			"/CreationDate": now,
		},
	}
}

func (d *Document) AllocID() ObjectID {
	id := d.nextID
	d.nextID++
	return id
}

func (d *Document) AddObject(data interface{}) ObjectID {
	id := d.AllocID()
	d.Objects = append(d.Objects, &Object{ID: id, Gen: 0, Data: data})
	return id
}

func (d *Document) AddObjectAt(id ObjectID, data interface{}) {
	d.Objects = append(d.Objects, &Object{ID: id, Gen: 0, Data: data})
	if id >= d.nextID {
		d.nextID = id + 1
	}
}

func (d *Document) SetCatalog(objID ObjectID) {
	d.catalogRef = objID
}

func (d *Document) SetPagesRoot(objID ObjectID) {
	d.pagesRootRef = objID
}

func (d *Document) SetTrailerInfo(info map[string]interface{}) {
	d.TrailerInfo = info
}

func (d *Document) HasMode(mode Mode) bool {
	return d.Mode&mode != 0
}

func (d *Document) Build() []byte {
	enc := write.NewEncoder()
	enc.WriteHeader()

	sorted := make([]*Object, len(d.Objects))
	copy(sorted, d.Objects)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID < sorted[j].ID
	})

	maxID := ObjectID(0)
	for _, obj := range sorted {
		if obj.ID > maxID {
			maxID = obj.ID
		}
	}

	objOffsets := make(map[ObjectID]int64)

	for _, obj := range sorted {
		objOffsets[obj.ID] = int64(enc.Len())
		fmt.Fprintf(enc, "%d %d obj\n", obj.ID, obj.Gen)

		switch data := obj.Data.(type) {
		case map[string]interface{}:
			enc.WriteDict(data)
			enc.WriteString("\n")
		case *write.Stream:
			dict := data.Dict
			if dict == nil {
				dict = make(map[string]interface{})
			}
			if _, ok := dict["/Length"]; !ok {
				dict["/Length"] = len(data.Data)
			}
			enc.WriteStream(dict, data.Data)
			enc.WriteString("\n")
		case []byte:
			_, _ = enc.Write(data)
			enc.WriteString("\n")
		default:
			fmt.Fprintf(enc, "%v\n", data)
		}

		enc.WriteString("endobj\n")
	}

	// Optional Info dict object (omitted for PDF/A-4)
	var infoRef ObjectID
	if d.TrailerInfo != nil && !d.HasMode(ModePDFA4) {
		infoRef = maxID + 1
		objOffsets[infoRef] = int64(enc.Len())
		fmt.Fprintf(enc, "%d 0 obj\n", infoRef)
		enc.WriteDict(d.TrailerInfo)
		enc.WriteString("\n")
		enc.WriteString("endobj\n")
		maxID = infoRef
	}

	xrefOffset := int64(enc.Len())

	offsets := make([]int64, int(maxID)+1)
	for _, obj := range sorted {
		offsets[obj.ID] = objOffsets[obj.ID]
	}
	if infoRef > 0 {
		offsets[infoRef] = objOffsets[infoRef]
	}

	enc.WriteXref(offsets)

	h := sha256.Sum256([]byte("gocorepdfengine"))
	fileID := write.HexString(h[:16])

	enc.WriteString("trailer\n")
	trailerDict := map[string]interface{}{
		"/Size": int(maxID) + 1,
		"/Root": write.Ref(int(d.catalogRef), 0),
		"/ID":   []interface{}{fileID, fileID},
	}
	if infoRef > 0 {
		trailerDict["/Info"] = write.Ref(int(infoRef), 0)
	}
	enc.WriteDict(trailerDict)
	enc.WriteString("\n")

	enc.WriteStartXref(xrefOffset)

	enc.WriteEOF()

	return enc.Bytes()
}
