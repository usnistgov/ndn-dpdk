package iface

import (
	"io"

	"github.com/usnistgov/ndn-dpdk/core/emission"
)

var emitter = emission.NewEmitter()

const (
	evt_FaceNew = iota
	evt_FaceUp
	evt_FaceDown
	evt_FaceClosing
	evt_FaceClosed
	evt_RxGroupAdd
	evt_RxGroupRemove
)

type FaceIdEventCallback func(faceId FaceId)

type RxGroupEventCallback func(rxg IRxGroup)

// Register a callback when a new face is created.
// Return a Closer that cancels the callback registration.
func OnFaceNew(cb FaceIdEventCallback) io.Closer {
	return emitter.On(evt_FaceNew, cb)
}

// Register a callback when a face becomes UP.
// Return a Closer that cancels the callback registration.
func OnFaceUp(cb FaceIdEventCallback) io.Closer {
	return emitter.On(evt_FaceUp, cb)
}

// Register a callback when a face becomes DOWN.
// Return a Closer that cancels the callback registration.
func OnFaceDown(cb FaceIdEventCallback) io.Closer {
	return emitter.On(evt_FaceDown, cb)
}

// Register a callback when a face is closing.
// Return a Closer that cancels the callback registration.
func OnFaceClosing(cb FaceIdEventCallback) io.Closer {
	return emitter.On(evt_FaceClosing, cb)
}

// Register a callback when a face is closed.
// Return a Closer that cancels the callback registration.
func OnFaceClosed(cb FaceIdEventCallback) io.Closer {
	return emitter.On(evt_FaceClosed, cb)
}

// Register a callback when an RxGroup is added.
// Return a Closer that cancels the callback registration.
func OnRxGroupAdd(cb RxGroupEventCallback) io.Closer {
	return emitter.On(evt_RxGroupAdd, cb)
}

// Register a callback when an RxGroup is removed.
// Return a Closer that cancels the callback registration.
func OnRxGroupRemove(cb RxGroupEventCallback) io.Closer {
	return emitter.On(evt_RxGroupRemove, cb)
}

func EmitRxGroupAdd(rxg IRxGroup) {
	emitter.EmitSync(evt_RxGroupAdd, rxg)
}

func EmitRxGroupRemove(rxg IRxGroup) {
	emitter.EmitSync(evt_RxGroupRemove, rxg)
}
