# ndn-dpdk/iface/socketface

This package implements a face using socket as transport.

**socketFace** type represents a socket face.
See [face creation](../../docs/face.md) "socket face" section for locator syntax.

The underlying transport and redial logic are implemented in [socketransport](../../ndn/sockettransport) package.
This package copies packets between `[]byte` of the underlying transport and DPDK's mbufs.
