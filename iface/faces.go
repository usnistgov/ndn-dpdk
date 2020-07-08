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
		log.Panic("invalid ID", face)
	}
	if gFaces[id] != nil {
		log.Panic("duplicate ID", face)
	}
	gFaces[id] = face
	emitter.EmitSync(evtFaceNew, id)
	ActivateTxFace(face)
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

// CloseAll closes all faces, RxLoops, and TxLoops.
func CloseAll() {
	for _, face := range List() {
		face.Close()
	}
	for _, rxl := range ListRxLoops() {
		rxl.Close()
	}
	for _, txl := range ListTxLoops() {
		txl.Close()
	}
}
