# ndn-dpdk/iface/ethport

This package implements faces using DPDK ethdev as transport.

**Face** type represents an Ethernet-based face or a memif face.
See [package ethface](../ethface) and [package memifface](../memifface) for more information.

**Port** type organizes faces on the same DPDK ethdev.
It manages ethdev resources and prevents conflicts among the faces.

## Receive Path

There are three receive path implementations.
One of them is chosen during port creation; the choice cannot be changed afterwards.

**RxFlow** is a hardware-accelerated receive path.
It uses one or more RX queues per face, and creates a *flow* via rte\_flow API to steer incoming frames to those queues.
The hardware performs header matching; there is minimal checking on software side.

**RxTable** is a software receive path.
It continuously polls ethdev RX queue 0 for incoming frames.
For each incoming frame, the software performs header matching (implemented in `EthRxMatch` struct), and then labels each matched frame with the face ID.
If no match is found for an incoming frame, it is dropped.

**RxMemif** is a memif-specific receive path, where each port has only one face.
It continuously polls ethdev RX queue 0 for incoming frames, and then labels each frame with the only face ID.

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev TX queue 0.
It prepends Ethernet/UDP/VXLAN headers to each frame (implemented in `EthTxHdr` struct), and requires every outgoing packet to have sufficient headroom for the headers.

The send path is thread-safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Therefore, **iface.TxLoop** calls `EthFace_TxBurst` from the same thread for all faces on the same port.
