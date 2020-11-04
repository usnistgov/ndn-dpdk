# ndn-dpdk/iface/ethface

This package implements Ethernet faces using DPDK ethdev as transport.

**ethFace** type represents an Ethernet face.
Locator of an Ethernet face has the following fields:

* *scheme* is set to "ether".
* *local* and *remote* are MAC-48 addresses written in the six groups of two lower-case hexadecimal digits separated by colons.
* *local* must be a unicast address.
* *remote* may be unicast or multicast.
  Every face is assumed to be point-to-point, even when using a multicast remote address.
* *vlan* (optional) is an VLAN ID in the range 0x001-0xFFF.
* *port* (optional) is the port name as presented by DPDK.
  If omitted, *local* is used to search for a suitable port; if specified, this takes priority over *local*.

**Port** type organizes faces on the same DPDK ethdev.
Each port can have zero or one face with multicast remote address, and zero or more faces with unicast remote addresses.
Faces on the same port can be created and destroyed individually.

## Receive Path

There are two receive path implementations.
All faces on the same port must use the same receive path implementation.

**rxFlow** type implements a hardware-accelerated receive path.
It uses one RX queue per face, and creates an rte\_flow to steer incoming frames to that queue.
An incoming frame is accepted only if it has the correct MAC addresses and VLAN tag.
There is minimal checking on software side.

**rxTable** type implements a software receive path.
It continuously polls ethdev RX queue 0 for incoming frames.
Header of an incoming frame is matched against each face, and labeled with the matching face ID.
If no match is found, drop the frame.

Port/face setup procedure is dominated by the choice of receive path implementation.
Initially, the port attempts to operate with rxFlows.
This can fail if the ethdev does not support rte\_flow, does not support the specific rte\_flow features used in `EthFace_SetupFlow`, or has fewer RX queues than the number of requested faces.
If rxFlows fail to setup for these or any other reason, the port falls back to rxTable.

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev TX queue 0.
It requires every outgoing packet to have sufficient headroom for the Ethernet header.

The send path is thread-safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Normally, **iface.TxLoop** invokes `EthFace_TxBurst` from the same thread.

## Memif Face

Shared memory packet interface (memif) is supported through this package.

Locator of a memif face has the following fields:

* *scheme* is set to "memif".
* *socketName* is the control socket filename.
  It must be an absolute path not exceeding 108 characters.
* *id* is the interface identifier in the range 0x00000000-0xFFFFFFFF.

In the data plane:

* Application must operate its memif interface in "slave" mode.
* Each packet must be an Ethernet frame carrying an NDNLPv2 frame.
* Application must use Ethernet address `F2:6C:E6:8D:9E:34`.
* NDN-DPDK uses Ethernet address `F2:71:7E:76:5D:1C`.
