# ndn-dpdk/iface/socketface

This package implements a face using socket as transport.

**socketFace** type represents a socket face.
Its Locator has the following fields:

* *scheme* is one of "udp", "tcp", "unix".
* *remote* is an address string acceptable to Go [net.Dial](https://golang.org/pkg/net/#Dial) function.
* *local* (optional) has the same format as *remote*, and is accepted only with "udp" scheme.

The underlying transport and redial logic are implemented in [socketransport](../../ndn/sockettransport) package.
This package copies packets between `[]byte` of the underlying transport and DPDK's mbufs.
