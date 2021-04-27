package ethdev

import (
	"github.com/usnistgov/ndn-dpdk/core/events"
)

var detachEmitter = events.NewEmitter()

// OnDetach registers a callback when a port is stopped and detached.
// Return a Closer that cancels the callback registration.
func OnDetach(dev EthDev, cb func()) (cancel func()) {
	return detachEmitter.Once(dev.ID(), cb)
}
