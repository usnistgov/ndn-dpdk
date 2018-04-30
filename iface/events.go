package iface

import (
	"github.com/chuckpreslar/emission"
)

type EventCallback func(faceId FaceId)

// Register a callback when a new face is created.
func OnFaceNew(cb EventCallback) {
	emitter.On(evt_FaceNew, cb)
}

// Register a callback when a face becomes UP.
func OnFaceUp(cb EventCallback) {
	emitter.On(evt_FaceUp, cb)
}

// Register a callback when a face becomes DOWN.
func OnFaceDown(cb EventCallback) {
	emitter.On(evt_FaceDown, cb)
}

// Register a callback when a face is closed.
func OnFaceClosed(cb EventCallback) {
	emitter.On(evt_FaceClosed, cb)
}

const (
	evt_FaceNew = iota
	evt_FaceUp
	evt_FaceDown
	evt_FaceClosed
)

var emitter = emission.NewEmitter()
