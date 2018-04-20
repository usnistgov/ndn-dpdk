# ndn-dpdk/iface/socketface

This package implements a face using socket as transport.

There are two different implementations:

* **datagramImpl** for datagram-oriented sockets (net.PacketConn).
* **streamImpl** for stream-oriented sockets (net.Conn that is not net.PacketConn).

FaceId of SocketFace is randomly assigned from the range 0xE000-0xEFFF.
LocalUri and RemoteUri reflect local and remote endpoint addresses, except that IPv6 addresses will be shown as "192.0.2.6", and non-UDP/TCP endpoints will be shown as "192.0.2.0:1".

SocketFace's send path is thread safe.

## Receive Path

A goroutine running `impl.RxLoop` function reads from the socket, and places L2 frames on the `SocketFace.rxQueue` channel.

**datagramImpl** assumes each incoming datagram is an L2 frame.
It casts DPDK mbuf's internal buffer as a `[]byte`, and does not copy the frame bytes.

**streamImpl** reads the incoming stream into a `[]byte`, extracts completed TLV elements with `ndn.TlvBytes.ExtractElement` function, and copies them to DPDK mbufs for posting to the channel.

Calling code must run `RxGroup.RxLoop` in an LCore to retrieve L2 frames from the `SocketFace.rxQueue` channel and pass them to `FaceImpl_RxBurst`.

## Send Path

The transmission function provided in `Face.txBurstOp` is `go_SocketFace_TxBurst`.
It places outgoing L2 frames on the `SocketFace.txQueue` channel.
A goroutine running `SocketFace.txLoop` function retrieves from the channel, and passes them to `impl.Send`.
In most cases, DPDK mbuf's internal buffer is casted as a `[]byte`, and does not need copying; however, `datagramImpl.Send` would have to copy if the mbuf is segmented.
