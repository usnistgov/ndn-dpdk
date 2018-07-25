# ndn-dpdk/iface

This package implements the face system, which provides network interfaces (faces) that can send and receive NDN packets.
Each face has a **FaceId**, a uint16 number that identifies the face.

There are three kinds of lower layer implementations:

* [EthFace](ethface/) communicates on Ethernet via DPDK ethdev.
* [SocketFace](socketface/) communicates on Unix/TCP/UDP tunnels via Go sockets.
* [MockFace](mockface/) is for unit testing.

Unit tests of this package are in [ifacetest](ifacetest/) subdirectory.

## Face System API

In C, public APIs are defined in term of **FaceId**.
There are functions to query face status, and to transmit a burst of packets.
Notably, there isn't a function to receive packets; instead, each lower layer implementation offers an "RX loop" function for receiving packets.

In Go, **IFace** interface defines what methods a face must provide.
Each lower layer implementation offers a `New` function that creates an instance that implements IFace interface.
That instance should embed **BaseFace** struct that implements many methods required by IFace.
`Get` function retrieves an existing IFace by FaceId; `IterFaces` enumerates all faces.

## Receive Path

The receive path starts from an "RX loop" function offered by lower layer implementations.
The RX loop continually retrieves L2 frames from one or more faces, and passes a received burst of L2 frames to `FaceImpl_RxBurst`.

`FaceImpl_RxBurst` first calls **RxProc** to decode L2 frames into L3 packets.
It then passes a burst of L3 packets to a **Face\_RxCb** callback provided by the user of face system (such as forwarder's input function).

RxProc is thread safe as long as different "RxProc thread number" is being used.
Currently, only thread 0 is capable of NDNLP reassembly.

## Send Path

The send path starts from `Face_TxBurst` function.
It first calls **TxProc** to encode L3 packets into L2 frames.
It then passes a burst of L2 frames to a **FaceImpl\_TxBurst** function provided by the lower layer implementation.

TxProc is normally not thread safe.
It can be made thread safe by `EnableThreadSafeTx` function that adds an output queue.
The face must then join a **TxLooper** that dequeues and sends packets.
This package provides two variants of TxLooper: **SingleTxLoop** for a single high-traffic face, and **MultiTxLoop** for multiple low-traffic faces (slower due to use of RCU).

## NDNLPv2

RxProc and TxProc partially implement [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2) indexed fragmentation feature.
One limitation is that the reassembler cannot handle out-of-order arrival.
