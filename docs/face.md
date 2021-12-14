# NDN-DPDK Face Creation

In [ICN terminology](https://www.rfc-editor.org/rfc/rfc8793.html#section-3.2-1), a **face** is a generalization of the network interface that can represent a physical network interface, an overlay inter-node channel, or an intra-node inter-process communication channel to an application.
NDN-DPDK supports the latter two categories, where each face represents an adjacency that communicates with one peer entity.
This page explains how to create faces in NDN-DPDK.

Face creation parameters are described with **locator**, a JSON document that conforms to the JSON schema `locator.schema.json` (installed in `/usr/local/share/ndn-dpdk` and [available online](https://ndn-dpdk.ndn.today/schema/locator.schema.json)).
A locator contains the transport protocol, local and remote endpoint addresses, and other related parameters.

After activating NDN-DPDK service, each role offers a different API that accepts a locator for face creation.
Read [forwarder](forwarder.md), [traffic generator](trafficgen.md), and [file server](fileserver.md) for details.

In any role, you can retrieve a list of faces with `ndndpdk-ctrl list-face` command or programmatically via GraphQL `faces` query.
The return value would contain the locator of each existing face.

## Ethernet-based Face

An Ethernet-based face communicates with a remote node on an Ethernet adapter using a DPDK networking driver.
It supports Ethernet (with optional VLAN header), UDP, and VXLAN protocols.
Its implementation is in [package ethface](../iface/ethface).

There are two steps in creating an Ethernet-based face:

1. Create an Ethernet port on the desired Ethernet adapter.
2. Create an Ethernet-based face on the Ethernet port.

Each Ethernet adapter can have multiple Ethernet-based faces.
The **Ethernet port** organizes those faces on the same adapter.
During port creation, sufficient hardware resources are reserved to accommodate anticipated faces on the adapter, and the adapter becomes ready for face creation.

There are three kinds of drivers for Ethernet port creation.
The following table gives a basic comparison:

driver kind | speed | supported hardware | Ethernet | VLAN | UDP | VXLAN | main limitation
-|-|-|-|-|-|-|-
PCI | fastest | some | yes | yes | yes | yes | exclusive NIC control
XDP | fast | all | yes | yes | port 6363 | no | MTU≤3300
AF\_PACKET | slow | all | yes | no | no | no |

The most suitable port creation command is hardware dependent, and some trial-and-error may be necessary.
Due to limitations in DPDK drivers, a failed port creation command may cause DPDK to enter an inconsistent state.
Therefore, before trying to a different port creation command, it is recommended to restart the NDN-DPDK service and redo the activation step.

### Ethernet Port with PCI Driver

DPDK offers user space drivers for [a range of network interface cards](https://doc.dpdk.org/guides/nics/).
If you have a supported NIC, you should create the Ethernet port with PCI driver, which enables the best performance.

Generally, to create an Ethernet port with PCI driver, you should:

1. Determine the PCI address of the Ethernet adapter.
2. Bind the PCI device to the proper kernel driver as expected by the DPDK driver.
3. Run `ndndpdk-ctrl create-eth-port` command with `--pci` flag.

Some Ethernet adapters support rte\_flow API that allows for a hardware-accelerated receive path called *RxFlow*.
See [package ethface](../iface/ethface) "Receive Path" section for detailed explanation.
You may enable this feature with `--rx-flow` flag, which substantially improves performance.
The specified number of queues is the maximum number of faces you can create on the Ethernet port.
Enabling RxFlow on a NIC that does not support it causes either port creation failure or face creation failure.

Example commands:

```bash
# list PCI addresses
dpdk-devbind.py --status-dev net

# change kernel driver binding (only needed for some NICs, see DPDK docs on what driver to use)
sudo dpdk-devbind.py -b uio_pci_generic 04:00.0

# create an Ethernet port with PCI driver, enabling RxFlow with 16 queues
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500 --rx-flow 16

# or, create an Ethernet port with PCI driver, disabling RxFlow
ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 1500
```

See [hardware known to work](hardware.md) page for instructions and examples on select NIC models.

Creating an Ethernet port with PCI driver causes DPDK to assume exclusive control over the PCI device.
After that, it is not possible to run other traffic (such as IP) over the same Ethernet adapter.
If you are connected over SSH, please ensure the SSH session does not rely on this network interface.

### Ethernet Port with XDP Driver

If DPDK does not have a PCI driver for your NIC or you encounter errors while using the PCI driver, you may create the Ethernet port with XDP driver.
This driver communicates with the Ethernet adapter via AF\_XDP socket, optimized for high performance packet processing.

To create an Ethernet port with XDP driver, you should:

1. Ensure the network interface is "up" and visible to the NDN-DPDK service process.
2. Run `ndndpdk-ctrl create-eth-port` command with `--netif` and `--xdp` flags.

Example commands:

```bash
NETIF=eth1

# if NDN-DPDK is running on the host: bring up the network interface
sudo ip link set $NETIF up

# if NDN-DPDK is running in a Docker container:
# (1) move the network interface into the container's network namespace
sudo ip link set $NETIF netns $(docker inspect --format='{{ .State.Pid }}' ndndpdk-svc)
# (2) bring up the network interface
docker exec ndndpdk-svc ip link set $NETIF up

# create an Ethernet port with XDP driver
ndndpdk-ctrl create-eth-port --netif $NETIF --xdp --mtu 1500
```

XDP driver is installed only if the libbpf library is installed before building DPDK.
If you have installed dependencies with the `ndndpdk-depends.sh` script, libbpf is installed automatically.

Due to kernel limitation, MTU is limited to about 3300 octets.
Setting an unacceptable MTU causes port creation failure.

During XDP driver activation, the Ethernet adapter is configured to have only 1 RX channel and RX-VLAN offload is disabled, and then an XDP program is loaded.
The XDP program recognizes NDN over Ethernet (with optional VLAN header) and NDN over IPv4/IPv6 + UDP on port 6363; it does not recognize VXLAN or other UDP ports.
If you need VXLAN, you can create a kernel interface with `ip link add` command, and create an Ethernet port on that network interface.

### Ethernet Port using AF\_PACKET Driver

If neither PCI driver nor XDP driver can be used, as a last resort you may use the AF\_PACKET driver.
This driver communicates with the Ethernet adapter via AF\_PACKET socket, which is substantially slower than the other two options.
To create an Ethernet port with XDP driver, you should:

1. Ensure the network interface is "up" and visible to the NDN-DPDK service process.
2. Run `ndndpdk-ctrl create-eth-port` command with `--netif` flag.

Example commands:

```bash
# see previous section for how to bring up the interface and make it visible to the NDN-DPDK service process

# create an Ethernet port with AF_PACKET driver
ndndpdk-ctrl create-eth-port --netif $NETIF --mtu 9000
```

An Ethernet port with AF\_PACKET only works reliably for NDN over Ethernet (without VLAN header).
While it is possible to create VLAN, UDP, or VXLAN faces on such as port, they may trigger undesirable reactions from the kernel network stack (e.g. ICMP port unreachable packets or UFW drop logs), because the kernel is unaware of NDN-DPDK's UDP endpoint.

### Creating Ethernet-based Face

After creating an Ethernet port, you can create Ethernet-based faces on the adapter.

Locator of an Ethernet face has the following fields:

* *scheme* is set to "ether".
* *local* and *remote* are MAC-48 addresses written in the six groups of two lower-case hexadecimal digits separated by colons.
* *local* must be a unicast address.
* *remote* may be unicast or multicast.
  Every face is assumed to be point-to-point, even when using a multicast remote address.
* *vlan* (optional) is an VLAN ID in the range 0x001-0xFFF.
  If omitted, the packets do not have VLAN header.
* *port* (optional) is the port name as presented by DPDK.
  If omitted, *local* is used to search for a suitable port; if specified, this takes priority over *local*.

Locator of a UDP tunnel face has the following fields:

* *scheme* is set to "udpe".
* All fields in "ether" locator are inherited.
* Both *local* and *remote* MAC addresses must be unicast.
* *localIP* and *remoteIP* are local and remote IP addresses.
  They may be either IPv4 or IPv6, and must be unicast.
* *localUDP* and *remoteUDP* are local and remote UDP port numbers.

Locator of a VXLAN tunnel face has the following fields:

* *scheme* is set to "vxlan".
* All fields in "udpe" locator, except *localUDP* and *remoteUDP*, are inherited.
* UDP destination port number is fixed to 4789; source port is random.
* *vxlan* is the VXLAN Network Identifier.
* *innerLocal* and *innerRemote* are unicast MAC addresses for inner Ethernet header.
* *maxRxQueues* (optional) is the maximum number of RX queues.
  When the Ethernet port is using PCI driver and has RxFlow enabled, setting this to greater than 1 could alleviate the bottleneck in forwarder's input thread.
  However, it would take up multiple RX queues as specified in `--rx-flow` flag during port creation.

See [package ethface](../iface/ethface) "UDP and VXLAN tunnel face" section for caveats, limitations, and what faces can coexist on the same port.

## Memif Face

A memif face communicates with a local application via [shared memory packet interface (memif)](https://docs.fd.io/vpp/21.10/dc/dea/libmemif_doc.html).
Its implementation is in [package ethface](../iface/ethface).
Although memif is implemented as an Ethernet device, you do not need to create an Ethernet port for the memif device.

Locator of a memif face has the following fields:

* *scheme* is set to "memif".
* *role* is either "server" or "client".
  It's recommended to use "server" role on NDN-DPDK side and "client" role on application side.
* *socketName* is the control socket filename.
  It must be an absolute path not exceeding 108 characters.
* *id* is the interface identifier in the range 0x00000000-0xFFFFFFFF.
* *socketOwner* may be set to a tuple `[uid,gid]` to change owner uid:gid of the control socket.
  It would allow applications to connect to NDN-DPDK without running as root.
  This currently works with libmemif but not gomemif, so that NDNgo still needs to run as root.

## Socket Face

A socket face communicates with either a local application or a remote entity via TCP/IP sockets.
It supports UDP, TCP, and Unix stream.
Its implementation is in [package ethface](../iface/socketface).

Locator of a socket face has the following fields:

* *scheme* is one of "udp", "tcp", "unix".
* *remote* is an address string acceptable to Go [net.Dial](https://pkg.go.dev/net#Dial) function.
* *local* (optional) has the same format as *remote*, and is accepted only with "udp" scheme.

You may have noticed that UDP is supported both as an Ethernet-based face and as a socket face.
The differences are:

* Ethernet-based UDP face ("udpe" scheme) runs on a DPDK Ethernet device without going through the kernel network stack.
  It is fast but does not follow normal IP routing and cannot communicate with local applications.
* Socket UDP face ("udp" scheme) goes through the kernel network stack.
  It behaves like a normal IP application but is much slower.

## Troubleshooting

### Error during Ethernet Port Creation or Face Creation

If the command or GraphQL mutation for creating an Ethernet port or a face returns an error, you can typically gather additional error information from NDN-DPDK service logs.
See [installation guide](INSTALL.md) "usage" section and [Docker container](Docker.md) "control the service container" section for how to access NDN-DPDK service logs.

Common mistakes include:

* Trying to create a face before activating NDN-DPDK service.
* NDN-DPDK is running in Docker but the Docker container was started with insufficient privileges and bind mounts.
* Requesting a higher MTU than what's allowed by `.mempool.DIRECT.dataroom` of the activation parameter.
* Requesting a driver kind or parameter on an Ethernet adapter that doesn't support it.

### Face Created but No Packet Received

Problem scenario is similar as the ndnping sample of [NDN-DPDK forwarder](forwarder.md):

* You have two forwarders A and B connected over a network link.
* The consumer program is connected to forwarder A over memif.
* The producer program is connected to forwarder B over memif.
* Faces were created without immediate errors.
* You expect the consumer program to receive Data packets, but it does not.

```text
|--------|  memif  |-----------|  Ethernet  |-----------|  memif  |--------|
|consumer|---------|forwarder A|------------|forwarder B|---------|producer|
|--------|         |-----------|            |-----------|         |--------|

faces   (1)       (2)         (3)          (4)         (5)       (6)
```

General steps to troubleshoot this issue is:

1. Confirm that all the faces exist.

   Run `ndndpdk-ctrl list-face` command on both forwarders.
   You should see faces (2) (3) (4) (5).

   Ethernet faces (3) and (4) are meant to be created manually.
   Some applications, such as `ndndpdk-godemo`, can automatically create memif faces (2) and (5); other applications, such as the NDN-DPDK fileserver, require you to manually create the memif faces.

2. Confirm that the FIB entries exist and point to the correct nexthops.

   Run `ndndpdk-ctrl list-fib` command on both forwarders.
   You should see a FIB entry on each forwarder for the producer's name prefix, pointing to face (3) and (5) respectively.

3. Locate where does packet loss start through face counters.

   Run `ndndpdk-ctrl get-face --cnt --id` *FACE-ID* command to retrieve counters of a face.
   Wait a few seconds and run this command again, and you can observe which counters are increasing.

   Follow the traffic flow in the order below, and locate the first counter that is not increasing:

   1. (1) TX-Interest, (2) RX-Interest, (3) TX-Interest, (4) RX-Interest, (5) TX-Interest, (6) RX-Interest.
   2. (6) TX-Data, (5) RX-Data, (4) TX-Data, (3) RX-Data, (2) TX-Data, (1) RX-Data.

   If the loss starts at an RX counter, some possible causes are:

   * Mismatched face locator.
   * Incorrect MTU configuration, such as MTU larger than what the link supports.
   * Driver issue/limitation, such as trying to use VXLAN on an Ethernet port with XDP driver.
   * Insufficient DIRECT mempool capacity: if the DIRECT mempool is full, DPDK silently drops all incoming packets.
     See [performance tuning](tuning.md) "memory usage insights" for how to see mempool utilization.
   * Packet corruption along the network link.
     If this occurs on face (4) RX-Interest, you can stop NDN-DPDK, capture traffic with `tcpdump`, and analyze the packet trace.

   If the loss starts at an TX counter, some possible causes are:

   * Missing FIB entry.
   * For face (1) and (6): The consumer/producer is not sending Interests/Data as you expected.
   * For face (4): the Data sent by producer are not satisfying the Interests according to the NDN protocol.

   In case an application does not publish face counters, you can skip to the next counter, but then you need to consider the possible causes for the previous step.
   For example, if you cannot retrieve face (1) TX-Interest counter from the consumer, and you see face (2) RX-Interest not increasing, you should consider "consumer not sending Interests" as a possible cause.

### Combination of Multiple Ethernet-based Faces Not Working

Problem scenario:

* You want to create two or more Ethernet-based faces on the same Ethernet port.
* When you have only one face on the Ethernet port, it works perfectly.
* After you create the second (or third, etc) face on the same Ethernet port, face creation fails with error, or some faces cannot send or receive traffic.

This is probably a DPDK driver limitation.
It most frequently occurs when using the PCI driver and enabling RxFlow.

You can try these options, one at a time (restart NDN-DPDK before trying another):

* Increase number of queues for RxFlow.
* Decrease number of RX queues in VXLAN locator.
* Disable RxFlow.
* Switch to XDP or AF\_PACKET driver.
