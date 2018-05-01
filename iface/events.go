package iface

import (
	"io"

	"github.com/chuckpreslar/emission"
)

var emitter = emission.NewEmitter()

const (
	evt_FaceNew = iota
	evt_FaceUp
	evt_FaceDown
	evt_FaceClosing
	evt_FaceClosed
)

type EventCallback func(faceId FaceId)

type eventCanceler struct {
	evt int
	cb  EventCallback
}

func (c eventCanceler) Close() error {
	emitter.Off(c.evt, c.cb)
	return nil
}

// Register a callback when a new face is created.
// Return a Closer that cancels the callback registration.
func OnFaceNew(cb EventCallback) io.Closer {
	emitter.On(evt_FaceNew, cb)
	return eventCanceler{evt_FaceNew, cb}
}

// Register a callback when a face becomes UP.
// Return a Closer that cancels the callback registration.
func OnFaceUp(cb EventCallback) io.Closer {
	emitter.On(evt_FaceUp, cb)
	return eventCanceler{evt_FaceUp, cb}
}

// Register a callback when a face becomes DOWN.
// Return a Closer that cancels the callback registration.
func OnFaceDown(cb EventCallback) io.Closer {
	emitter.On(evt_FaceDown, cb)
	return eventCanceler{evt_FaceDown, cb}
}

// Register a callback when a face is closing.
// Return a Closer that cancels the callback registration.
func OnFaceClosing(cb EventCallback) io.Closer {
	emitter.On(evt_FaceClosing, cb)
	return eventCanceler{evt_FaceClosing, cb}
}

// Register a callback when a face is closed.
// Return a Closer that cancels the callback registration.
func OnFaceClosed(cb EventCallback) io.Closer {
	emitter.On(evt_FaceClosed, cb)
	return eventCanceler{evt_FaceClosed, cb}
}
