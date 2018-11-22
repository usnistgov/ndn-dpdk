# ndn-dpdk/iface/mockface

This package implements a mock face for unit testing.

Test code can invoke `MockFace.Rx` to cause the face to receive a packet.
These packets are queued in `iface.ChanRxGroup`.
Calling code must run `iface.ChanRxGroup` in an LCore to receive these packets.

Packets transmitted through a mock face are accumulated on `MockFace.TxInterests`, `MockFace.TxData`, or `MockFace.TxNacks` slices.
Test code is responsible for freeing these packets.

FaceId of MockFace is randomly assigned from the range 0x0001-0x0FFF.
LocalUri and RemoteUri are both "mock:".

MockFace's send path is not thread safe.
