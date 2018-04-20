# ndn-dpdk/iface/mockface

This package implements a mock face for unit testing.

Test code can invoke `MockFace.Rx` to cause the face to receive a packet.
All MockFaces depend on `MockFace.TheRxLoop` singleton as their `iface.IRxLooper`.

Packets transmitted through a mock face are accumulated on `MockFace.TxInterests`, `MockFace.TxData`, or `MockFace.TxNacks` slices.
Test code is responsible for freeing these packets.

FaceId of MockFace is randomly assigned from the range 0x0001-0x0FFF.
LocalUri and RemoteUri are both "mock://".

MockFace's send path is not thread safe.
