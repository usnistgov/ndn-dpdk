# ndn-dpdk/core/emission

This package is a thin wrapper of [github.com/chuckpreslar/emission](https://godoc.org/github.com/chuckpreslar/emission).
It modifies `emitter.On` method to return an `io.Closer` for canceling the callback registration.
