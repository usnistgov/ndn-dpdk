# Hardware Known to Work with NDN-DPDK

NDN-DPDK works with a number of hardware devices.
This page lists some hardware known to work with NDN-DPDK.
Note that this is not a complete list.

> Certain commercial entities, equipment, or materials may be identified in this document in order to describe an experimental procedure or concept adequately.
> Such identification is not intended to imply recommendation or endorsement by the National Institute of Standards and Technology, nor is it intended to imply that the entities, materials, or equipment are necessarily the best available for the purpose.

## CPU and Memory

NDN-DPDK only works on x86\_64 (amd64) architecture.
See [DPDK getting started guide for Linux](https://doc.dpdk.org/guides/linux_gsg/) for system requirements of DPDK.
In particular, SSE 4.2 instruction set is required.

The developers have tested NDN-DPDK on servers with one, two, and four NUMA sockets.

Default configuration of NDN-DPDK requires at least 6 CPU cores (total) and 8 GB hugepages memory (per NUMA socket).
With a custom configuration, NDN-DPDK might work on 2 CPU cores and 2 GB memory, albeit at reduced performance; see [performance tuning](tuning.md) "lcore allocation" and "memory usage insights" for some hints on how to do so.

Additionally, you should have at least 1 GB memory on NUMA socket 0 and on each NUMA socket where you have PCI device(s) that you want to use.
DPDK device drivers are not well-tested when they cannot allocate memory on NUMA socket 0 or the NUMA socket of the PCI device.
In that case, you would see "Cannot allocate memory" error in NDN-DPDK service logs.

## Ethernet Adapters

NDN-DPDK aims to work with most Ethernet adapters supported by [DPDK Network Interface Controller Drivers](https://doc.dpdk.org/guides/nics/).

The developers have tested NDN-DPDK with the following Ethernet adapters:

model | speed | DPDK driver
-|-|-
Mellanox ConnectX-5 | 100 Gbps | mlx5
Intel X710 | 10 Gbps | i40e, iavf
Intel X520 | 10 Gbps | ixgbe
Intel I350 | 1 Gbps | igb

Some Ethernet adapters have more than one physical ports on the same PCI card.
NDN-DPDK is only tested to work on the first port (lowest PCI address) of those dual-port or quad-port adapters.
If you encounter face creation failure or you are unable to send/receive packets, please use the first port instead.

### Mellanox Ethernet Adapters

DPDK supports Mellanox adapters in Ethernet mode, but not in Infiniband mode.
If you have VPI adapters, use `mlxconfig` tool to verify and change port mode.
See [MLX5 poll mode driver](https://doc.dpdk.org/guides/nics/mlx5.html) for more information.

The libibverbs library must be installed before building DPDK or running the `ndndpdk-depends.sh` script:

```bash
sudo apt install libibverbs-dev

# for building Docker image
docker build \
  --build-arg APT_PKGS="libibverbs-dev"
  [other arguments]
```

To use Mellanox adapters in Docker container, add these flags when you launch the service container:

```bash
docker run \
  --device /dev/infiniband \
  --mount type=bind,source=/sys,target=/sys \
  [other arguments]
```

* `--device /dev/infiniband` flag enables access to IB verbs device.
* `--mount target=/sys` flag enables access to hardware counters in `/sys/class/net` directory.

It's also necessary to move the kernel network interface into the container's network namespace.
This needs to be performed every time the container is started or restarted, before activating NDN-DPDK.
Forgetting this step would cause `Unable to recognize master/representors on the multiple IB devices` error in container logs, and the device would be unusable.
Example command:

```bash
NETIF=enp4s0f0
sudo ip link set $NETIF netns $(docker inspect --format='{{ .State.Pid }}' ndndpdk-svc)
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
sudo dpdk-devbind.py -b igb_uio 04:00.0
```

To use Intel adapters in Docker container, add these flags when you launch the service container:

```bash
docker run \
  $(find /dev -name 'uio*' -type c -printf ' --device %p') \
  --mount type=bind,source=/sys,target=/sys \
  [other arguments]
```

* `find` subcommand constructs `--device` flags for `/dev/uio*` devices.
* `--mount target=/sys` flag enables access to attributes in `/sys/class/uio` directory.

Some Intel Ethernet adapters support network virtual functions (VFs).
It allows sharing the same physical adapter between NDN-DPDK and kernel IP stack, either on the physical server or with NDN-DPDK in a virtual machine.
To use Intel VF:

1. Create a VF using the procedure given in [Intel SR-IOV Configuration Guide](https://www.intel.com/content/dam/www/public/us/en/documents/technology-briefs/xl710-sr-iov-config-guide-gbe-linux-brief.pdf) "Server Setup" section.

    * You must assign a valid unicast MAC address to the VF.
      Example command:

      ```bash
      # generate a random locally administered unicast MAC address
      HWADDR=$(openssl rand -hex 6 | sed -E 's/../:\0/g;s/^:(.)./\12/')

      # assign the MAC address to the VF; 'enp4s0f0' refers to the physical adapter
      sudo ip link set enp4s0f0 vf 0 mac $HWADDR

      # verify the settings: you should see a non-zero MAC address on 'vf 0' line
      ip link show enp4s0f0
      ```

    * Face creation commands should use the MAC address assigned above, not the MAC address of the physical adapter.

2. If NDN-DPDK is installed on the physical server: bind the VF PCI device to igb\_uio driver.

3. If NDN-DPDK is installed in a virtual machine: passthrough the VF PCI device to the virtual machine, bind the VF to igb\_uio driver in the guest OS.

### AF\_XDP and AF\_PACKET Sockets

NDN-DPDK can work with any Ethernet adapter supported by the Linux kernel via DPDK net\_af\_xdp and net\_af\_packet socket drivers.
This allows the use of Ethernet adapters not supported by DPDK PCI drivers.
However, these socket-based drivers have limited functionality and lower performance.

To use socket-based drivers, the Ethernet adapter must be "up" and visible to the NDN-DPDK service process.
Example command to ensure these conditions:

```bash
NETIF=eth1

# if NDN-DPDK is running on the host: bring up the network interface
sudo ip link set $NETIF up

# if NDN-DPDK is running in a Docker container:
# (1) move the network interface into the container's network namespace
sudo ip link set $NETIF netns $(docker inspect --format='{{ .State.Pid }}' ndndpdk-svc)
# (2) bring up the network interface
docker exec ndndpdk-svc ip link set $NETIF up
```

During face creation, if the Ethernet adapter has not been activated with a DPDK PCI driver, NDN-DPDK will attempt to activate it with a socket-based driver.
It is unnecessary to manually define a DPDK virtual device in activation parameters.

The **net\_af\_xdp** driver uses AF\_XDP sockets, optimized for high performance packet processing.
This driver requires Linux kernel ≥5.4.
The libbpf library must be installed before building DPDK; the `ndndpdk-depends.sh` script installs libbpf automatically if a compatible kernel version is found.
Due to kernel limitation, MTU is limited to about 3300 octets; setting an unacceptable MTU causes port activation failure.

During net\_af\_xdp activation, the Ethernet adapter is configured to have only 1 RX channel and RX-VLAN offload is disabled, and then an XDP program is loaded.
The XDP program recognizes NDN over Ethernet and NDN over IPv4/IPv6 + UDP on port 6363; it does not recognize VXLAN or other UDP ports.
If you need VXLAN, you can create a kernel interface with `ip link add` command, and create the face on that network interface.

The **net\_af\_packet** driver uses AF\_PACKET sockets.
This is compatible with older kernels, but it is substantially slower.

It's recommended to use net\_af\_packet for NDN over Ethernet (no VLAN) only.
Trying to use VLAN, UDP, and VXLAN may trigger undesirable reactions from the kernel network stack (e.g. ICMP port unreachable packets or UFW drop logs), because the kernel is unaware of NDN-DPDK's UDP endpoint.

In case you encounter problems during face creation on an AF\_XDP virtual device, you may disable AF\_XDP and force AF\_PACKET by setting `.vdevConfig.xdp.disabled=true` on the locator, such as:

```jsonc
{
  "scheme": "ether", // also supported on "udpe" and "vxlan"
  "vdevConfig": {
    "xdp": { "disabled": true }
  }
  // other fields
}
```
