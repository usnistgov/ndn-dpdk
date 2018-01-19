# ndn-dpdk/iface

This package implements the face system.

Unit tests of this package are in [ifacetest](ifacetest/) subdirectory.

## Face

`Face` represents a network interface that can send and receive NDN packets.

`TxProc` and `RxProc` implement the send path and the receive path, respectively.
They translate between network layer packets and link layer packets.

Each lower layer implementation (known as "Transport" in NFD) provides a `FaceOps` structure when creating a face.
Lower layer actions, such as transmitting and receiving L2 packets, are delegated to functions provided in this structure.

## NDNLP Support

`TxProc` and `RxProc` implement a subset of [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2) features.

**Fragmentation and reassembly**: reassembler cannot handle out-of-order arrival.

Other features are not implemented.

## FaceTable

`FaceTable` type stores a pointer to each face.

Each inserted face must have a unique FaceId.
Each lower layer implementation is allocated a range of FaceIds, and they are responsible for allocating the FaceId.
