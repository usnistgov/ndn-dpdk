# ndn-dpdk/iface

This package implements the face system.

Unit tests of this package are in [ifacetest](ifacetest/) subdirectory.

## Face

**Face** represents a network interface that can send and receive NDN packets.

**RxProc** and **TxProc** implement the receive path and the send path, respectively.
They translate between network layer packets and link layer packets.
They also implement [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2) fragmentation-reassembly feature, but the reassembler cannot handle out-of-order arrival.

Each lower layer implementation (in NFD they are known as "Transports") provides a number of function pointers for lower layer actions, such as transmitting L2 frames and closing the face.
They are either contained in a **FaceOps** struct, or placed on the **Face** struct directly.

Notably, lower layer implementations do not supply a function pointer for receiving packets.
Instead, they offer a "RX loop" function that continually retrieves L2 frames from one or more faces, and passes a received burst of L2 frames to `FaceImpl_RxBurst`, which in turn passes them to **RxProc**.
All "RX loop" functions must accept a **Face\_RxCb** callback, which would be invoked when a burst of L3 packets arrives.
The receive path is not thread safe; as such, only one "RX loop" should receive for a face.

`Face_TxBurst` starts the send path.
The send path is normally not thread safe.
It can be made thread safe by `Face.EnableThreadSafeTx` function that adds a queue.
The face must then join a **TxLooper** that dequeues and sends packets.

## FaceTable

**FaceTable** type stores a pointer to each face.

Each inserted face must have a unique FaceId.
Each lower layer implementation is allocated a range of FaceIds, and they are responsible for allocating the FaceId.

FaceTable is thread safe.
