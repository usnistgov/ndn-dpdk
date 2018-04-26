package iface

var gFaces [int(FACEID_MAX) + 1]IFace

// Get face by FaceId.
func Get(faceId FaceId) IFace {
	return gFaces[faceId]
}

// Put face (non-thread-safe).
// This should be called by face subtype constructor.
func Put(face IFace) {
	faceId := face.GetFaceId()
	if faceId.GetKind() == FaceKind_None {
		panic("invalid FaceId")
	}
	if gFaces[faceId] != nil {
		panic("duplicate FaceId")
	}
	gFaces[faceId] = face
}

// Iterator over faces.
//
// Usage:
// for it := iface.IterFaces(); it.Valid(); it.Next() {
//   // use it.Id and it.Face
// }
type FaceIterator struct {
	Id   FaceId
	Face IFace
}

func IterFaces() *FaceIterator {
	var it FaceIterator
	it.Id = FACEID_INVALID
	it.Next()
	return &it
}

func (it *FaceIterator) Valid() bool {
	return it.Id <= FACEID_MAX
}

func (it *FaceIterator) Next() {
	for it.Id++; it.Id <= FACEID_MAX; it.Id++ {
		it.Face = gFaces[it.Id]
		if it.Face != nil {
			return
		}
	}
	it.Face = nil
}

// Close all faces.
func CloseAll() {
	for it := IterFaces(); it.Valid(); it.Next() {
		it.Face.Close()
	}
}
