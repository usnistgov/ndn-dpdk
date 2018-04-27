# ndn-dpdk/iface/socketface

This package implements a face using socket as transport.

FaceId of SocketFace is randomly assigned from the range 0xE000-0xEFFF.
LocalUri and RemoteUri reflect local and remote endpoint addresses, except that IPv6 addresses become "192.0.2.6", and unknown Unix endpoints become "/invalid".

## Receive Path

A goroutine running `impl.RxLoop` function reads from the socket, and places L2 frames on the `SocketFace.rxQueue` channel.

On a datagram-oriented socket, each incoming datagram is an L2 frame.
The implementation casts DPDK mbuf's internal buffer as a `[]byte`, and does not copy the frame bytes.

On a stream-oriented socket, the implementation reads the incoming stream into a `[]byte`, extracts completed TLV elements with `ndn.TlvBytes.ExtractElement` function, and copies them to DPDK mbufs.

Calling code must run `RxGroup.RxLoop` in an LCore to retrieve L2 frames from the `SocketFace.rxQueue` channel and pass them to `FaceImpl_RxBurst`.

## Send Path

The transmission function provided in `Face.txBurstOp` is `go_SocketFace_TxBurst`.
It places outgoing L2 frames on the `SocketFace.txQueue` channel.

A goroutine running `SocketFace.txLoop` function then retrieves framesfrom the `SocketFace.txQueue` channel, and passes them to `impl.Send`.
In most cases, DPDK mbuf's internal buffer is casted as a `[]byte`, and does not need copying; however, sending a segmented mbuf to a datagram-oriented socket requires copying.

The send path is thread safe.

## Error Handling

If Read or Write on the net.Conn returns an error, `Face.handleError` processes the error as follows:

* Errors during face closing cause the RxLoop or txLoop to stop.
* Temporary net.Error is ignored.
* Otherwise, the net.Conn is redialed until it reconnects successfully or the face is closed.

For UDP and TCP, the same local address is retained during redialing.
