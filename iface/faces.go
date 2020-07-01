package iface

var gFaces [MaxID + 1]Face

// Get retrieves face by ID.
// Returns nil if id is invalid.
func Get(id ID) Face {
	if !id.Valid() {
		return nil
	}
	return gFaces[id]
}

// Put stores face.
// This is non-thread-safe.
// This should be called by face subtype constructor.
func Put(face Face) {
	id := face.ID()
	if !id.Valid() {
		panic("invalid ID")
	}
	if gFaces[id] != nil {
		panic("duplicate ID")
	}
	gFaces[id] = face
	emitter.EmitSync(evtFaceNew, id)
}

// List returns a list of existing faces.
func List() (list []Face) {
	for _, face := range gFaces {
		if face != nil {
			list = append(list, face)
		}
	}
	return list
}

// CloseAll closes all faces.
func CloseAll() {
	for _, face := range List() {
		face.Close()
	}
}
