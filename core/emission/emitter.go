package emission

import (
	"io"

	chuckpreslar_emission "github.com/chuckpreslar/emission"
)

// Emitter is a simple event emitter.
type Emitter struct {
	*chuckpreslar_emission.Emitter
}

// NewEmitter creates a simple event emitter.
func NewEmitter() (emitter *Emitter) {
	emitter = new(Emitter)
	emitter.Emitter = chuckpreslar_emission.NewEmitter()
	return emitter
}

// On registers a callback when an event occurs.
// Returns an io.Closer that cancels the callback registration.
func (emitter *Emitter) On(event, listener interface{}) io.Closer {
	emitter.Emitter.On(event, listener)
	return canceler{emitter.Emitter, event, listener}
}

type canceler struct {
	emitter  *chuckpreslar_emission.Emitter
	event    interface{}
	listener interface{}
}

func (c canceler) Close() error {
	c.emitter.Off(c.event, c.listener)
	return nil
}
