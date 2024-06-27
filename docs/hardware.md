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

model | speed | DPDK driver | RxFlow Ethernet | RxFlow UDP | RxFlow VXLAN | RxFlow GTP-U
-|-|-|-|-|-|-
NVIDIA ConnectX-5 | 100 Gbps | mlx5 | yes | yes | yes | no
NVIDIA ConnectX-6 | 200 Gbps | mlx5 | yes | yes | yes | yes
Intel X710 | 10 Gbps | i40e | no | yes | no | yes
Intel X710 VF | 10 Gbps | iavf | untested | untested | untested | untested
Intel XXV710 | 25 Gbps | i40e | untested | untested | untested | untested
Intel X520 | 10 Gbps | ixgbe | no | yes | no | untested
Intel I350 | 1 Gbps | igb | no | no | no | untested
Broadcom/QLogic 57810 | 10 Gbps | bnx2x | untested | untested | untested | untested

Some Ethernet adapters have more than one physical ports on the same PCI card.
NDN-DPDK is only tested to work on the first port (lowest PCI address) of those dual-port or quad-port adapters.
If you encounter face creation failure or you are unable to send/receive packets, please use the first port instead.

See [face creation](face.md) for the general procedure of face creation on Ethernet adapters.
The next sections provide information on specific NIC models.

### NVIDIA Ethernet Adapters

DPDK supports NVIDIA adapters in Ethernet mode, but not in Infiniband mode.
If you have VPI adapters, use `mlxconfig` tool to verify and change port mode.
See [MLX5 common driver](https://doc.dpdk.org/guides/platform/mlx5.html) for more information.

#### Setup with bifurcated driver

The libibverbs library must be installed before building DPDK or running the `ndndpdk-depends.sh` script:

```bash
sudo apt install libibverbs-dev

# for building Docker image
docker build \
  --build-arg APT_PKGS="libibverbs-dev" \
  [other arguments]
```

NVIDIA adapters use a bifurcated driver.
You should not change PCI driver binding with `dpdk-devbind.py` command.

To use NVIDIA adapters in Docker container, add these flags when you launch the service container:

```bash
docker run \
  --device /dev/infiniband \
  --mount type=bind,source=/sys,target=/sys,readonly=true \
  [other arguments]
```

* `--device /dev/infiniband` flag enables access to IB verbs device.
* `--mount target=/sys` flag enables read-only access to Infiniband device hardware counters.

It's also necessary to move the kernel network interface into the container's network namespace.
This needs to be performed every time the container is started or restarted, before activating NDN-DPDK.
Forgetting this step causes port creation failure, and you would see `Unable to recognize master/representors on the multiple IB devices` error in container logs.
Example command:

```bash
NETIF=enp4s0f0
CTPID=$(docker inspect -f '{{.State.Pid}}' ndndpdk-svc)
sudo ip link set $NETIF netns $CTPID
```

To use NVIDIA adapters in a KVM guest, passthrough the PCI device to the virtual machine.
When creating the port, if you encounter `mlx5_common: Verbs device not found` error (seen in NDN-DPDK service logs), verify that mlx5\_ib kernel module is loaded.
On Ubuntu, you can install `linux-image-generic` package (in place of `linux-image-virtual` found in some cloud images) to obtain this kernel module.

#### RxFlow Feature

Most NVIDIA adapters are compatible with NDN-DPDK RxFlow feature.
Unless you encounter errors, you should enable RxFlow while creating the Ethernet port.
Example command:

```bash
# determine maximum number of RxFlow queues
# look at "current hardware settings - combined" row
ethtool --show-channels enp4s0f0

# create Ethernet port
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500 --rx-flow 16
```

GTP-U tunnel face with RxFlow is supported on ConnectX-6 and above.
Moreover, *GTP flow matching* feature shall be enabled in the firmware.
See [MLX5 common driver](https://doc.dpdk.org/guides/platform/mlx5.html) regarding `FLEX_PARSER_PROFILE_ENABLE=3` parameter.
Otherwise, GTP-U face creation on RxFlow would fail with "GTP support is not enabled" log message.

### Intel Ethernet Adapters

Intel Ethernet adapters may be used with either vfio-pci or igb\_uio driver.
See [Memory in DPDK Part 2: Deep Dive into IOVA](https://www.intel.com/content/www/us/en/developer/articles/technical/memory-in-dpdk-part-2-deep-dive-into-iova.html) for their differences.
Generally, it's recommended to use VFIO.

#### Setup with VFIO

The vfio-pci driver requires IOMMU.
Follow [Arch Linux PCI passthrough guide](https://wiki.archlinux.org/title/PCI_passthrough_via_OVMF#Setting_up_IOMMU) for how to enable IOMMU.

With IOMMU enabled, to bind a PCI device to vfio-pci:

```bash
sudo modprobe vfio-pci
sudo dpdk-devbind.py -b vfio-pci 04:00.0
```

To use Intel adapters with VFIO in Docker container, the driver binding must still be performed on the host.
Then, add these flags when you launch the NDN-DPDK service container:

```bash
docker run \
  --device /dev/vfio \
  [other arguments]
```

* `--device /dev/vfio` flag enables access to VFIO devices.
* Driver binding must be configured before starting the container.
  Otherwise, port creation fails and you would see `Failed to open VFIO group` error in container logs.

#### Setup with UIO

The igb\_uio driver is not recommended, but it may be used if IOMMU does not work.
IOMMU must be disabled or set to passthrough mode with `iommu=pt` kernel parameter for igb\_uio to work.

The igb\_uio driver is available from [dpdk-kmods repository](https://git.dpdk.org/dpdk-kmods/tree/).
To install the driver:

```bash
git clone https://dpdk.org/git/dpdk-kmods
cd dpdk-kmods/linux/igb_uio
make clean all
UIODIR=/lib/modules/$(uname -r)/kernel/drivers/uio
sudo install -d -m0755 $UIODIR
sudo install -m0644 igb_uio.ko $UIODIR
sudo depmod
```

To bind a PCI device to igb\_uio:

```bash
sudo modprobe igb_uio
sudo dpdk-devbind.py -b igb_uio 04:00.0
```

The igb\_uio driver requires DPDK to operate in *IOVA as Physical Addresses (PA) Mode*.
To choose this mode, set **.eal.iovaMode** to "PA" in NDN-DPDK activation parameters.

To use Intel adapters with UIO in Docker container, add these flags when you launch the NDN-DPDK service container:

```bash
docker run \
  --privileged \
  [other arguments]
```

* `--privileged` flag enables writing to PCI device config.
  See also [moby issue #22825](https://github.com/moby/moby/issues/22825).

#### Virtual Function

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

2. If NDN-DPDK is installed on the physical server: bind the VF PCI device to vfio-pci driver.

3. If NDN-DPDK is installed in a virtual machine: passthrough the VF PCI device to the virtual machine, bind the VF to vfio-pci driver in the guest OS.

#### RxFlow Feature

Intel adapters have limited compatibility with NDN-DPDK RxFlow feature.
You should test RxFlow with your specific hardware and face locators, and decide whether to use this feature.
Example command:

```bash
# determine maximum number of RxFlow queues
# look at "current hardware settings - combined" row
# you can only run this command before binding to vfio-pci driver
ethtool --show-channels enp4s0f0

# create Ethernet port, enable RxFlow
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500 --rx-flow 16

# or, create Ethernet port, disable RxFlow
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500
```

GTP-U tunnel face with RxFlow is supported with [I40E poll mode driver](https://doc.dpdk.org/guides/nics/i40e.html) on Intel Ethernet 700 series.
It relies on [Dynamic Device Personalization (DDP)](https://www.intel.com/content/www/us/en/developer/articles/technical/dynamic-device-personalization-for-intel-ethernet-700-series.html) feature.
You must manually download the *GTPv1 DDP profile* and place it at `/lib/firmware/intel/i40e/ddp/gtp.pkg`.
If the profile is found, you would see "upload DDP package success" log message during Ethernet port creation.
Without the profile, GTP-U face creation on RxFlow would fail with "GTP is not supported by default" log message.
During NDN-DPDK service shutdown, a profile rollback will be attempted.
In case of an abnormal shutdown, you may need to power-cycle the server to cleanup the profile.

### Broadcom/QLogic Ethernet Adapters

BCM57810 has been tested with igb\_uio driver.
The setup procedure is very similar to Intel Ethernet adapters with UIO.

To use Broadcom/QLogic adapters with UIO in Docker container, add these flags when you launch the NDN-DPDK service container:

```bash
docker run \
  --privileged \
  --mount type=bind,source=/lib/firmware,target=/lib/firmware,readonly=true \
  [other arguments]
```

* `--privileged` flag enables writing to PCI device config.
* `--mount target=/lib/firmware` flag enables read-only access to firmware files.

### Ethernet Adapters Not Supported by DPDK

NDN-DPDK can work with any Ethernet adapter supported by the Linux kernel via XDP and AF\_PACKET drivers.
This allows the use of Ethernet adapters not supported by DPDK PCI drivers.
See [face creation](face.md) for how to create an Ethernet port with these drivers.

## NVMe Storage Device

NDN-DPDK can use NVMe storage device as forwarder Content Store expansion.
It can work with most NVMe devices.

Best efficiency is achieved on a NVMe controller that supports "scatter-gather lists" (SGLs) feature.
Run `nvme id-ctrl -H /dev/nvme0` and look at "sgls" field to identify this feature.
