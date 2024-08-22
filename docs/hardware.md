# Hardware Known to Work with NDN-DPDK

NDN-DPDK works with a number of hardware devices.
This page lists some hardware known to work with NDN-DPDK.
Note that this is not a complete list.

## CPU and Memory

NDN-DPDK only works on x86\_64 (amd64) architecture.
See [DPDK getting started guide for Linux](https://doc.dpdk.org/guides/linux_gsg/) for system requirements of DPDK.
In particular, SSE 4.2 instruction set is required.

The developers have tested NDN-DPDK on servers with one, two, and four NUMA sockets.

Default configuration of NDN-DPDK requires at least 6 CPU cores (total) and 8 GB hugepages memory (per NUMA socket).
With a custom configuration, NDN-DPDK might work on 2 CPU cores and 2 GB memory, albeit at reduced performance; see [performance tuning](tuning.md) "lcore allocation" and "memory usage insights" for some hints on how to do so.

Generally, you should allow at least 1 GB memory on NUMA socket 0 and on each NUMA socket where you have Ethernet adapters that you want to use with PCI driver.
DPDK device drivers are not well-tested when they cannot allocate memory on NUMA socket 0 or the NUMA socket of the PCI device.
In that case, you would see "Cannot allocate memory" error in NDN-DPDK service logs.

## Ethernet Adapters

See [Ethernet adapters known to work](nics.md).

## NVMe Storage Device

NDN-DPDK can use NVMe storage device as forwarder Content Store expansion.
It can work with most NVMe devices.

Best efficiency is achieved on a NVMe controller that supports "scatter-gather lists" (SGLs) feature.
Run `nvme id-ctrl -H /dev/nvme0` and look at "sgls" field to identify this feature.
