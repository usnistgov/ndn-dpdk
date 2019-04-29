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
Notably, there isn't a function to receive packets; instead, RxLoop type is used for receiving packets.

In Go, **IFace** type defines what methods a face must provide.
Each lower layer implementation offers functions to create an instance that implements IFace interface.
That instance should embed **FaceBase** struct that implements many methods required by IFace.
`Get` function retrieves an existing IFace by FaceId; `IterFaces` enumerates all faces.

All faces are assumed to be point-to-point.
**Locator** type identifies the endpoints of a face.
It has a `Scheme` field that indicates the underlying network protocol, as well as other fields added by each lower layer implementation.
This type can be marshaled as JSON and YAML.

## Receive Path

**RxLoop** type implements the receive path.
Lower layer implementation places each face into one or more **RxGroup**s, which are then added into RxLoops.
`RxLoop_Run` function continually invokes `RxGroup.rxBurstOp` function to retrieves L2 frames.
It then passes a burst of L2 frames to `FaceImpl_RxBurst`.

`FaceImpl_RxBurst` first calls **RxProc** to decode L2 frames into L3 packets.
It then passes a burst of L3 packets to a **Face\_RxCb** callback provided by the user of face system (such as forwarder's input function).
RxProc is thread safe as long as different "RxProc thread number" is being used.
Currently, only thread 0 is capable of NDNLP reassembly.

## Send Path

The send path starts from `Face_TxBurst` function.
It enqueues a burst of L3 packets in `Face.txQueue` (the "before-Tx queue").

**TxLoop** type implements the send path.
It dequeues a burst of L3 packets from `Face.txQueue`, calls **TxProc** to encode them into L2 frames.
It then passes a burst of L2 frames to a **FaceImpl\_TxBurst** function provided by the lower layer implementation.

## NDNLPv2

RxProc and TxProc partially implement [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2) indexed fragmentation feature.
One limitation is that the reassembler cannot handle out-of-order arrival.
