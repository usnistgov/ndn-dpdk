# ndn-dpdk/iface/ethface

This package implements Ethernet faces using DPDK ethdev as transport.

**EthFace** type represents an Ethernet face.
Its FaceId is randomly assigned from the range 0x1000-0x1FFF.
Its FaceUri has three parts:

*   User information portion contains the local or remote Ethernet address.
    It is a MAC-48 address, written as upper case hexadecimal, using hyphen to separate octets.
*   Hostname portion contains the port name as presented by DPDK.
    Characters other than alphanumeric and underscore are replaced by hyphens.
*   Port number portion contains the VLAN identifier (currently not supported).

Multiple EthFaces can co-exist on the same DPDK ethdev.
They are organized by the **Port** type.
Each Port can have zero or one multiple EthFace, and zero or more unicast EthFaces.
All EthFaces on the same Port must be created together.

## Receive Path

**EthRxFlow** type implements a hardware-accelerated receive path.
It requires the ethdev to support rte\_flow API.
It uses one RX queue per face, and creates a flow to steering incoming frames with matching MAC address to that queue.
There is minimal checking on software side.

**EthRxTable** type implements a software receive path as a fallback.
It polls RX queue 0, accepts all Ethernet frames with NDN EtherType, and discards all frames with non-NDN EtherType (such as VLAN-tagged frames, IP packets, and NDN packets over UDP/TCP tunnels).
Then, it labels each accepted frame with incoming FaceId: if the destination MAC address is a group address, the FaceId is set to the multicast face; otherwise, the last octet of source MAC address is looked up in a 256-element array of unicast FaceIds.
In case a face selected as above does not exist, the frame's incoming FaceId is set to `FACEID_INVALID`, so that `FaceImpl_RxBurst` would drop the frame.
Because of this dispatching procedure, EthRxTable requires every unicast EthFace on the same port to have distinct last octet in its remote MAC address.

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev queue 0.
It requires every outgoing packet to have sufficient headroom for the Ethernet header.

The send path is thread safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Otherwise, the caller must ensure that transmissions on all EthFaces of the same ethdev occur on the same thread.
