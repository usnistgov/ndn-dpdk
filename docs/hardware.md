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

Generally, you should allow at least 1 GB memory on NUMA socket 0 and on each NUMA socket where you have Ethernet adapters that you want to use with PCI driver.
DPDK device drivers are not well-tested when they cannot allocate memory on NUMA socket 0 or the NUMA socket of the PCI device.
In that case, you would see "Cannot allocate memory" error in NDN-DPDK service logs.

## Ethernet Adapters

NDN-DPDK aims to work with most Ethernet adapters supported by [DPDK Network Interface Controller Drivers](https://doc.dpdk.org/guides/nics/).

The developers have tested NDN-DPDK with the following Ethernet adapters:

model | speed | DPDK driver | RxFlow
-|-|-|-
Mellanox ConnectX-5 | 100 Gbps | mlx5 | yes
Intel X710 | 10 Gbps | i40e | UDP only
Intel X710 VF | 10 Gbps | iavf | untested
Intel X520 | 10 Gbps | ixgbe | UDP only
Intel I350 | 1 Gbps | igb | no

Some Ethernet adapters have more than one physical ports on the same PCI card.
NDN-DPDK is only tested to work on the first port (lowest PCI address) of those dual-port or quad-port adapters.
If you encounter face creation failure or you are unable to send/receive packets, please use the first port instead.

See [face creation](face.md) for the general procedure of face creation on Ethernet adapters.
The next sections provide information on specific NIC models.

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
Forgetting this step causes port creation failure, and you would see `Unable to recognize master/representors on the multiple IB devices` error in container logs.
Example command:

```bash
NETIF=enp4s0f0
CTPID=$(docker inspect -f '{{.State.Pid}}' ndndpdk-svc)
sudo ip link set $NETIF netns $CTPID
```

Most Mellanox adapters are compatible with NDN-DPDK RxFlow feature.
Unless you encounter errors, you should enable RxFlow while creating the Ethernet port.
Example command:

```bash
# determine maximum number of RxFlow queues
# look at "current hardware settings - combined" row
ethtool --show-channels enp4s0f0

# create Ethernet port
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500 --rx-flow 16
```

### Intel Ethernet Adapters

The PCI device must use igb\_uio driver, available in [dpdk-kmods repository](https://git.dpdk.org/dpdk-kmods).
To install this driver, first install kernel headers, then:

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

To use Intel adapters in Docker container, the driver must still be installed and loaded on the host.
Then, add these flags when you launch the NDN-DPDK service container:

```bash
docker run \
  $(find /dev -name 'uio*' -type c -printf ' --device %p') \
  --mount type=bind,source=/sys,target=/sys \
  [other arguments]
```

* `find` subcommand constructs `--device` flags for `/dev/uio*` devices.
* `--mount target=/sys` flag enables access to attributes in `/sys/class/uio` directory.

The igb\_uio driver expects "IOVA as Physical Addresses (PA)" mode.
If you encounter port activation failure, in NDN-DPDK activation parameters, set **.eal.iovaMode** to `"PA"` to force this mode.

Intel adapters have limited compatibility with NDN-DPDK RxFlow feature.
As tested with i40e and ixgbe drivers, RxFlow can be used with UDP faces, but not Ethernet or VXLAN faces.
You should test RxFlow with your specific hardware and face locators, and decide whether to use this feature.
Example command:

```bash
# determine maximum number of RxFlow queues
# look at "current hardware settings - combined" row
# you can only run this command before binding to igb_uio driver
ethtool --show-channels enp4s0f0

# create Ethernet port, enable RxFlow
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500 --rx-flow 16

# or, create Ethernet port, disable RxFlow
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500
```

### Intel Virtual Function

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

    * All faces must use this assigned MAC address as the local address.
      You cannot use the MAC address of the physical adapter or any other address.

2. If NDN-DPDK is installed on the physical server: bind the VF PCI device to igb\_uio driver.

3. If NDN-DPDK is installed in a virtual machine: passthrough the VF PCI device to the virtual machine, bind the VF to igb\_uio driver in the guest OS.

### Unsupported Ethernet Adapters

NDN-DPDK can work with any Ethernet adapter supported by the Linux kernel via XDP and AF\_PACKET drivers.
This allows the use of Ethernet adapters not supported by DPDK PCI drivers.
See [face creation](face.md) for how to create an Ethernet port with these drivers.
