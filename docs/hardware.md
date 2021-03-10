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

### Mellanox Ethernet Adapters

The libibverbs library must be installed before building DPDK or running the `ndndpdk-depends.sh` script:

```bash
sudo apt install libibverbs-dev

# for Docker installation
docker build \
  --build-arg APT_PKGS="libibverbs-dev"
  [other arguments]
```

To use Mellanox adapters in Docker container, add these `docker run` flags when you launch the service container:

```bash
docker run \
  --device /dev/infiniband --device /dev/vfio \
  [other arguments]
```

### Intel Ethernet Adapters

The PCI device must use igb\_uio driver.
The `ndndpdk-depends.sh` script can automatically install this kernel module if kernel headers are present.

If you have upgraded the kernel or you are using the Docker container, you can install the kernel module manually:

```bash
git clone https://dpdk.org/git/dpdk-kmods
cd dpdk-kmods/linux/igb_uio
make
UIODIR=/lib/modules/$(uname -r)/kernel/drivers/uio
sudo install -d -m0755 $UIODIR
sudo install -m0644 igb_uio.ko $UIODIR
sudo depmod
```

Example command to bind the PCI device to igb\_uio driver:

```bash
sudo modprobe igb_uio
sudo dpdk-devbind.sh -b igb_uio 04:00.0
```

To use Intel adapters in Docker container, add these `docker run` flags when you launch the service container:

```bash
docker run \
  $(find /dev -name 'uio*' -type c -printf ' --device %p') \
  --mount type=bind,source=/sys,target=/sys \
  [other arguments]
```

* `find` subcommand constructs `--device` flags for `/dev/uio*` devices.
* `--mount target=/sys` flag enables access to attributes in `/sys/class/uio` directory.
