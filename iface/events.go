package iface

import (
	"github.com/usnistgov/ndn-dpdk/core/events"
)

var emitter = events.NewEmitter()

const (
	evtFaceNew     = "FaceNew"
	evtFaceUp      = "FaceUp"
	evtFaceDown    = "FaceDown"
	evtFaceClosing = "FaceClosing"
	evtFaceClosed  = "FaceClosed"
)

// OnFaceNew registers a callback when a new face is created.
// Return a Closer that cancels the callback registration.
func OnFaceNew(cb func(ID)) (cancel func()) {
	return emitter.On(evtFaceNew, cb)
}

// OnFaceUp registers a callback when a face becomes UP.
// Return a Closer that cancels the callback registration.
func OnFaceUp(cb func(ID)) (cancel func()) {
	return emitter.On(evtFaceUp, cb)
}

// OnFaceDown registers a callback when a face becomes DOWN.
// Return a Closer that cancels the callback registration.
func OnFaceDown(cb func(ID)) (cancel func()) {
	return emitter.On(evtFaceDown, cb)
}

// OnFaceClosing registers a callback when a face is closing.
// Return a Closer that cancels the callback registration.
func OnFaceClosing(cb func(ID)) (cancel func()) {
	return emitter.On(evtFaceClosing, cb)
}

// OnFaceClosed registers a callback when a face is closed.
// Return a Closer that cancels the callback registration.
func OnFaceClosed(cb func(ID)) (cancel func()) {
	return emitter.On(evtFaceClosed, cb)
}
