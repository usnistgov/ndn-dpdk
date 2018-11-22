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
Currently, every unicast EthFace must have distinct last octet in its remote MAC address.
All EthFaces on the same Port must be created and destroyed together.

## Receive Path

**EthRxGroup** type implements the receive path.
Currently, the receive path only uses ethdev queue 0.
It polls one or more ethdevs, accepts all Ethernet frames with NDN EtherType, and discards all frames with non-NDN EtherType (such as VLAN-tagged frames, IP packets, and NDN packets over UDP/TCP tunnels).

Accepted frames are then labelled with incoming FaceIds and timestamp.
If a frame arrives on a non-existent face (e.g. unknown remote MAC address), its incoming FaceId is set to `FACEID_INVALID`, and `FaceImpl_RxBurst` would drop the frame.

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev queue 0.
It requires every outgoing packet to have sufficient headroom for the Ethernet header.

The send path is thread safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Otherwise, the caller must ensure that transmissions on all EthFaces of the same ethdev occur on the same thread.
