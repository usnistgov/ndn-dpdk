# ndn-dpdk/iface/ethface

This package implements a face using DPDK ethdev as transport.

Each face has dedicated control on an ethdev.
It only sends and receives NDN packets on Ethernet multicast.
It discards all frames with non-NDN EtherType, including NDN packets over UDP/TCP tunnels.

FaceId of EthFace is in the range 0x1000-0x1FFF, where the lower 12 bits is the ethdev's port number.
LocalUri indicates ethdev's MAC address.
RemoteUri indicates ethdev's name, as presented by DPDK; it does not show the actual remote address, which is a hard-coded multicast group address.

EthFace's send path is thread safe only if the underlying DPDK PMD is thread safe, which generally is not the case.

## Mbuf Usage

For each outgoing NDNLP packet, the send path allocates a mbuf for the Ethernet header and NDNLP header.
Payload of the NDNLP packet is chained via indirect mbufs.
