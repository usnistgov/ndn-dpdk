# ndn-dpdk/iface/socketface

This package implements a face using socket as transport.

**socketFace** type represents a socket face.
See [face creation](../../docs/face.md) "socket face" section for locator syntax.

## Generic Implementation

This package accepts any Go `net.Conn` instance, which could be either stream-oriented or datagram-oriented.
It reuses the functionality implemented in [socketransport](../../ndn/sockettransport) package.
That includes receiving and sending packets, and redialing failed stream-oriented sockets.

RX logic is implemented in **rxConns** type.
Each face has a goroutine that reads one packet at a time from the `sockettransport.Transport` into the dataroom of a DPDK mbuf.
Upon successfully receiving a packet, the mbuf is enqueued to a ring buffer, which is shared among all socket faces.
The RxLoop thread calls C `SocketRxConns_RxBurst` function to dequeue from this ring buffer.

TX logic is implemented in `go_SocketFace_TxBurst` function.
The TxLoop thread may call this function to write a packet to the `sockettransport.Transport`.
Since this is a synchronous call, the TxLoop thread could get blocked if the socket buffer is full.

## UDP Specialization

This package offers a specialized implementation when the `net.Conn` is identified to be a UDP socket.
Receiving and sending packets are implemented in C.

RX logic is implemented in **rxEpoll** type.
It maintains an [epoll](https://man7.org/linux/man-pages/man7/epoll.7.html) instance that waits for any of the socket file descriptors of all socket faces to become readable or have an error.
The RxLoop thread calls `SocketRxEpoll_RxBurst` function, which checks for epoll events.
Upon notified by epoll, packets are received from the socket into mbufs, and any socket errors are ignored.

TX logic is implemented in `SocketFace_DgramTxBurst` function, which transmits the packet via `sendmsg` syscall.
