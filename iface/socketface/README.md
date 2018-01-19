# ndn-dpdk/iface/socketface

This package implements a face using socket as transport.

Go code controls the socket.
Two goroutines are created for a socket:

* `impl.RxLoop` reads packets from the socket, and places them on a channel to be received by C callback.
* `SocketFace.txLoop` retrieves packets from a channel, and writes them onto the socket.

Several Go functions are exported as C callbacks, which are then provided in `FaceOps` structure.
