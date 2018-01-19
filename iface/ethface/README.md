# ndn-dpdk/iface/ethface

This package implements a face using DPDK ethdev as transport.

Each face has dedicated control on an ethdev.
It only sends and receives NDN packets on Ethernet multicast.
It discards all frames with non-NDN EtherType, including NDN packets over UDP/TCP tunnels.

## Mbuf Usage

For each outgoing NDNLP packet, the send path allocates a mbuf for the Ethernet header and NDNLP header.
Payload of the NDNLP packet is chained via indirect mbufs.
