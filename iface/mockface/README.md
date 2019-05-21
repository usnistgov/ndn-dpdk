# ndn-dpdk/iface/mockface

This package implements a mock face for unit testing.

**MockFace** type represents a mock face.
FaceId is randomly assigned from the range 0x0001-0x0FFF.
Locator has the following fields:

*   *Scheme* is set to "mock".

Test code can invoke `MockFace.Rx` to cause the face to receive a packet.
These packets are queued in `iface.ChanRxGroup`.
Calling code must add `iface.ChanRxGroup` to a TxLoop to receive these packets.

MockFace's send path is non-thread-safe.
Packets transmitted through a mock face are accumulated on `MockFace.TxInterests`, `MockFace.TxData`, or `MockFace.TxNacks` slices.
Test code is responsible for freeing these packets.
