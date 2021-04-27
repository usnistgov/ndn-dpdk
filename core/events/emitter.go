// Package events provides a simple event emitter.
package events

import (
	"github.com/tul/emission"
)

// Emitter is a simple event emitter.
// This is a thin wrapper of emission.Emitter that modifies emitter.On method to return a function that cancels the callback registration.
type Emitter struct {
	*emission.Emitter
}

// NewEmitter creates a simple event emitter.
func NewEmitter() *Emitter {
	return &Emitter{
		Emitter: emission.NewEmitter(),
	}
}

// On registers a callback when an event occurs.
// Returns a function that cancels the callback registration.
func (emitter *Emitter) On(event, listener interface{}) (cancel func()) {
	hdl := emitter.Emitter.On(event, listener)
	return emitter.makeCancel(event, hdl)
}

// Once registers a one-time callback when an event occurs.
// Returns a function that cancels the callback registration.
func (emitter *Emitter) Once(event, listener interface{}) (cancel func()) {
	hdl := emitter.Emitter.Once(event, listener)
	return emitter.makeCancel(event, hdl)
}

func (emitter *Emitter) makeCancel(event interface{}, hdl emission.ListenerHandle) func() {
	return func() { emitter.RemoveListener(event, hdl) }
}
