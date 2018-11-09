# ndn-dpdk/iface/ethface

This package implements a face using DPDK ethdev as transport.

FaceId of EthFace is in the range 0x1000-0x1FFF, where the lower 12 bits is the ethdev's port number.

FaceUri of EthFace has three parts:

*   User information portion contains the local or remote Ethernet address.
    It is a MAC-48 address, written as upper case hexadecimal, using hyphen to separate octets.
*   Hostname portion contains the port name as presented by DPDK.
    Characters other than alphanumeric and underscore are replaced by hyphens.
*   Port number portion contains the VLAN identifier.

Current implementation does not support non-default remote Ethernet address or VLAN identifier.

## Receive Path

`EthRxLoop` type implements the receive path.
It polls the ethdev, accepts all Ethernet frames with NDN EtherType, and discards all frames with non-NDN EtherType (such as VLAN-tagged frames, IP packets, and NDN packets over UDP/TCP tunnels).

Each RxLoop runs on a separate DPDK lcore.
It can receive on multiple faces, which allows a node to support a large number of low-traffic faces.
On the other hand, one may add the same face to multiple RxLoops to handle the workload on high-traffic faces.
In that case, each RxLoop must use a different RxProc thread number to avoid conflicts.

## Send Path

The send path requires every outgoing packet to have sufficient headroom for the Ethernet header.

The send path is thread safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Currently, the send path only uses queue 0.
