// Package events provides a simple event emitter.
package events

import (
	"io"

	"github.com/chuckpreslar/emission"
)

// Emitter is a simple event emitter.
// This is a thin wrapper of emission.Emitter that modifies emitter.On method to return an io.Closer that cancels the callback registration.
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
// Returns an io.Closer that cancels the callback registration.
func (emitter *Emitter) On(event, listener interface{}) io.Closer {
	emitter.Emitter.On(event, listener)
	return canceler{emitter.Emitter, event, listener}
}

// Once registers a one-time callback when an event occurs.
// Returns an io.Closer that cancels the callback registration.
func (emitter *Emitter) Once(event, listener interface{}) io.Closer {
	emitter.Emitter.Once(event, listener)
	return canceler{emitter.Emitter, event, listener}
}

type canceler struct {
	emitter  *emission.Emitter
	event    interface{}
	listener interface{}
}

func (c canceler) Close() error {
	c.emitter.Off(c.event, c.listener)
	return nil
}
