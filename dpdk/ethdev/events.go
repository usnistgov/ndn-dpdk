package ethdev

import (
	"github.com/usnistgov/ndn-dpdk/core/events"
)

var closeEmitter = events.NewEmitter()

// OnClose registers a callback when a port is stopped and closed.
// Returns a function that cancels the callback registration.
func OnClose(dev EthDev, cb func()) (cancel func()) {
	return closeEmitter.Once(dev.ID(), cb)
}
