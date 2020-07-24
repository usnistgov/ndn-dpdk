# ndn-dpdk/iface/ethface

This package implements Ethernet faces using DPDK ethdev as transport.

**ethFace** type represents an Ethernet face.
Its Locator has the following fields:

* *scheme* is set to "ether".
* *local* and *remote* are MAC-48 addresses written in the six groups of two lower-case hexadecimal digits separated by colons.
* *local* must be a unicast address.
* *remote* may be unicast or multicast.
  Every face is assumed to be point-to-point, even when using a multicast remote address.
* *vlan* (optional) is an VLAN ID in the range 0x001-0xFFF.
* *port* (optional) is the port name as presented by DPDK.
  If omitted, *local* is used to search for a suitable port; if specified, this takes priority over *local*.

**Port** type organizes EthFaces on the same DPDK ethdev.
Each port can have zero or one face with multicast remote address, and zero or more faces with unicast remote addresses.
EthFaces on the same port can be created and destroyed individually.

## Receive Path

There are two receive path implementations.
Currently, all faces on the same port must use the same receive path implementation.

**EthRxFlow** type implements a hardware-accelerated receive path.
It uses one RX queue per face, and creates an rte\_flow to steering incoming frames to that queue.
An incoming frame is accepted only if it has the correct MAC addresses and VLAN tag.
There is minimal checking on software side.

**EthRxTable** type implements a software receive path.
Its procedure is:

1. Poll ethdev RX queue 0 for incoming frames.
2. Label each frame with incoming ID:
    * If the destination MAC address is a group address, the ID is set to the face with multicast remote address.
    * Otherwise, the last octet of source MAC address is used to query a 256-element array of unicast FaceIDs.
      This requires every face with unicast remote address to have distinct last octet.
    * In case a face selected as above does not exist, the frame's incoming ID is set to zero.
      Later, `FaceImpl_RxBurst` would drop such a frame.
    * VLAN tags do not participate in packet dispatching.
3. Remove the Ethernet and VLAN headers.
   Drop the frame if it does not have the NDN EtherType (this includes NDN packets over UDP/TCP tunnels).

Port/face setup procedure is dominated by the choice of receive path implementation.
Initially, the port attempts to operate with EthRxFlows.
This can fail if the ethdev does not support rte\_flow, does not support the specific rte\_flow features used in `EthFace_SetupFlow`, or has fewer RX queues than the number of requested faces.
If EthRxFlows fail to setup for these or any other reason, the port falls back to EthRxTable.
It can fail if multiple faces with unicast remote addresses have the same last octet.
In case both receive path implementations fail to setup, the port would remain in an inoperational state.

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev TX queue 0.
It requires every outgoing packet to have sufficient headroom for the Ethernet header.

The send path is thread-safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Normally, **iface.TxLoop** invokes `EthFace_TxBurst` from the same thread.
