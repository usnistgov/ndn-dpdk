# Hardware Known to Work with NDN-DPDK

NDN-DPDK works with a number of hardware devices.
This page lists some hardware known to work with NDN-DPDK.
Note that this is not a complete list.

> Certain commercial entities, equipment, or materials may be identified in this document in order to describe an experimental procedure or concept adequately.
> Such identification is not intended to imply recommendation or endorsement by the National Institute of Standards and Technology, nor is it intended to imply that the entities, materials, or equipment are necessarily the best available for the purpose.

## CPU and Memory

NDN-DPDK only works on x86\_64 (amd64) architecture.
See [DPDK getting started guide for Linux](https://doc.dpdk.org/guides/linux_gsg/) for system requirements of DPDK.
In particular, SSE 3.2 instructions are required.

The developers have tested NDN-DPDK on servers with one, two, and four NUMA sockets.

Default configuration of NDN-DPDK requires at least 6 CPU cores (total) and 8 GB memory (per NUMA socket).
With a custom configuration, NDN-DPDK could work on 2 CPU cores and 2 GB memory, albeit at reduced performance.

## Ethernet Adapters

NDN-DPDK aims to work with most Ethernet adapters supported by [DPDK Network Interface Controller Drivers](https://doc.dpdk.org/guides/nics/).

The developers have tested NDN-DPDK with the following Ethernet adapters:

* Mellanox ConnectX-5, 100 Gbps, mlx5 driver
* Intel X710, 10 Gbps, i40e driver
* Intel X520, 10 Gbps, ixgbe driver
* Intel I350, 1 Gbps, igb driver

NDN-DPDK can also be used with DPDK [AF\_PACKET poll mode driver](https://doc.dpdk.org/guides/nics/af_packet.html) to support any Ethernet adapter, at reduced speeds.
