# ndn-dpdk/iface/faceuri

This package implements FaceUri, a string representation of network interface.

## FaceUri Syntax

* `dev://net_pcap0`: DPDK ethdev `net_pcap0`.
* `udp://10.0.2.1:6363`: UDP socket.
* `tcp://10.0.2.1:6363`: TCP socket.
* `mock://`: mock face for testing.
