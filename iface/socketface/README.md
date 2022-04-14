# ndn-dpdk/iface/socketface

This package implements a face using socket as transport.

**socketFace** type represents a socket face.
See [face creation](../../docs/face.md) "socket face" section for locator syntax.

The underlying transport and redial logic are implemented in [socketransport](../../ndn/sockettransport) package.
When receiving and sending packets, the dataroom in DPDK mbuf is converted to `[]byte` to achieve zero copy.
