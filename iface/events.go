package iface

import (
	"io"

	"github.com/usnistgov/ndn-dpdk/core/emission"
)

var emitter = emission.NewEmitter()

const (
	evtFaceNew = iota
	evtFaceUp
	evtFaceDown
	evtFaceClosing
	evtFaceClosed
	evtRxGroupAdd
	evtRxGroupRemove
)

// OnFaceNew registers a callback when a new face is created.
// Return a Closer that cancels the callback registration.
func OnFaceNew(cb func(ID)) io.Closer {
	return emitter.On(evtFaceNew, cb)
}

// OnFaceUp registers a callback when a face becomes UP.
// Return a Closer that cancels the callback registration.
func OnFaceUp(cb func(ID)) io.Closer {
	return emitter.On(evtFaceUp, cb)
}

// OnFaceDown registers a callback when a face becomes DOWN.
// Return a Closer that cancels the callback registration.
func OnFaceDown(cb func(ID)) io.Closer {
	return emitter.On(evtFaceDown, cb)
}

// OnFaceClosing registers a callback when a face is closing.
// Return a Closer that cancels the callback registration.
func OnFaceClosing(cb func(ID)) io.Closer {
	return emitter.On(evtFaceClosing, cb)
}

// OnFaceClosed registers a callback when a face is closed.
// Return a Closer that cancels the callback registration.
func OnFaceClosed(cb func(ID)) io.Closer {
	return emitter.On(evtFaceClosed, cb)
}

// OnRxGroupAdd registers a callback when an RxGroup is added.
// Return a Closer that cancels the callback registration.
func OnRxGroupAdd(cb func(RxGroup)) io.Closer {
	return emitter.On(evtRxGroupAdd, cb)
}

// OnRxGroupRemove registers a callback when an RxGroup is removed.
// Return a Closer that cancels the callback registration.
func OnRxGroupRemove(cb func(RxGroup)) io.Closer {
	return emitter.On(evtRxGroupRemove, cb)
}

// EmitRxGroupAdd emits the RxGroupAdd event.
func EmitRxGroupAdd(rxg RxGroup) {
	emitter.EmitSync(evtRxGroupAdd, rxg)
	ActivateRxGroup(rxg)
}

// EmitRxGroupRemove emits the RxGroupRemove event.
func EmitRxGroupRemove(rxg RxGroup) {
	emitter.EmitSync(evtRxGroupRemove, rxg)
	DeactivateRxGroup(rxg)
}
