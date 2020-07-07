# ndn-dpdk/iface

This package implements the face system, which provides network interfaces (faces) that can send and receive NDN packets.
Each face has a **ID**, a uint16 number that identifies the face.

There are three lower layer implementations:

* [EthFace](ethface/) communicates on Ethernet via DPDK ethdev.
* [SocketFace](socketface/) communicates on Unix/TCP/UDP tunnels via Go sockets.
* [MockFace](mockface/) is for unit testing.

Unit tests of this package are in [ifacetest](ifacetest/) subdirectory.

## Face System API

In C, public APIs are defined in term of **ID**.
There are functions to query face status, and to transmit a burst of packets.
Notably, there isn't a function to receive packets; instead, RxLoop type is used for receiving packets.

In Go, **Face** type defines what methods a face must provide.
Each lower layer implementation offers functions to create an instance that implements Face interface.
That instance should embed **FaceBase** struct that implements many methods required by Face.
`Get` function retrieves an existing Face by ID; `IterFaces` enumerates all faces.

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
It then passes a burst of L3 packets to the upper layer (such as forwarder's input function) via a **Face\_RxCb** callback.
RxProc is thread-safe as long as each thread uses a distinct "RxProc thread number".

## Send Path

The send path starts from `Face_TxBurst` function.
It enqueues a burst of L3 packets in `Face.txQueue` (the "before-Tx queue").
`Face_TxBurst` function is thread-safe.

**TxLoop** type implements the send path.
It dequeues a burst of L3 packets from `Face.txQueue`, calls **TxProc** to encode them into L2 frames.
It then passes a burst of L2 frames to the lower layer implementation via `Face.txBurstOp` function.
TxProc is non-thread-safe, so that only one thread should be running TxProc for a face.

## NDNLPv2

RxProc and TxProc partially implement [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2) indexed fragmentation feature.
The limitations are:

* When multiple threads are running RxProc on the same face, only "thread 0" can perform reassembly; fragments arriving on other threads are dropped.
* The reassembler cannot handle out-of-order arrival.

## Packet Queue

**PktQueue** type implements a packet queue that can operate in one of three modes.

*Plain* mode: a simple drop-tail queue.

*Delay* mode: a drop-tail queue that enforces a minimum amount of delay.
This is useful for simulating a processing delay.

*CoDel* mode: a queue that uses the [CoDel algorithm](https://tools.ietf.org/html/rfc8289).
This CoDel implementation differs from a standard implementation in that it dequeues packets in bursts instead of one at a time.
The last packet in each burst is used to calculate the sojourn time, and at most one packet can be dropped in each burst.
The `CoDel_*` functions are adapted from the CoDel implementation in the Linux kernel, under the BSD license (see [`codel.LICENSE`](../csrc/vendor/codel.LICENSE)).
