# ndn-dpdk/iface/ethface

This package implements Ethernet faces using DPDK ethdev as transport.

**ethFace** type represents an Ethernet face.
Locator of an Ethernet face has the following fields:

* *scheme* is set to "ether".
* *local* and *remote* are MAC-48 addresses written in the six groups of two lower-case hexadecimal digits separated by colons.
* *local* must be a unicast address.
* *remote* may be unicast or multicast.
  Every face is assumed to be point-to-point, even when using a multicast remote address.
* *vlan* (optional) is an VLAN ID in the range 0x001-0xFFF.
* *port* (optional) is the port name as presented by DPDK.
  If omitted, *local* is used to search for a suitable port; if specified, this takes priority over *local*.
* *portConfig* (optional) contains configuration for **Port** creation, considered on the first face on a port.
  See **PortConfig** type for details.

**Port** type organizes faces on the same DPDK ethdev.
Each port can have zero or one Ethernet face with multicast remote address, and zero or more Ethernet faces with unicast remote addresses.
Faces on the same port can be created and destroyed individually.

## Receive Path

There are three receive path implementations.
All faces on the same port must use the same receive path implementation.

**RxFlow** is a hardware-accelerated receive path.
It uses one RX queue per face, and creates an rte\_flow to steer incoming frames to that queue.
An incoming frame is accepted only if it has the correct MAC addresses and VLAN tag.
There is minimal checking on software side.

**RxTable** is a software receive path.
It continuously polls ethdev RX queue 0 for incoming frames.
Header of an incoming frame is matched against each face, and labeled with the matching face ID.
If no match is found, drop the frame.

**RxMemif** is a memif-specific receive path, where each port has only one face.
It continuously polls ethdev RX queue 0 for incoming frames, then labels each frame with the only face ID.

Port/face setup procedure is dominated by the choice of receive path implementation.
Initially, the port attempts to operate with rxFlows.
This can fail if the ethdev does not support rte\_flow, does not support the specific rte\_flow features used in `EthFace_SetupFlow`, or has fewer RX queues than the number of requested faces.
If rxFlows fail to setup for these or any other reason, the port falls back to rxTable.

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev TX queue 0.
It requires every outgoing packet to have sufficient headroom for the Ethernet header.

The send path is thread-safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Normally, **iface.TxLoop** invokes `EthFace_TxBurst` from the same thread.

## UDP and VXLAN Tunnel Face

UDP and VXLAN tunnels are supported through this package.

Locator of a UDP tunnel face has the following fields:

* *scheme* is set to "udpe".
  The suffix "e" means "ethface"; it is added to differentiate from the "udp" scheme implemented in socketface package.
* All fields in "ether" locator are inherited.
* Both *local* and *remote* MAC addresses must be unicast.
* *localIP* and *remoteIP* are local and remote IP addresses.
  They may be either IPv4 or IPv6, and must be unicast.
* *localUDP* and *remoteUDP* are local and remote UDP port numbers.

Locator of a VXLAN tunnel face has the following fields:

* *scheme* is set to "vxlan".
* *localIP* and *remoteIP* fields in "udpe" locator are inherited.
* UDP destination port number is fixed to 4789; source port is random.
* *vxlan* is the VXLAN Network Identifier.
* *innerLocal* and *innerRemote* are MAC addresses for inner Ethernet header.
* *maxRxQueues* (optional) is the maximum number of RX queues.
  When using rxFlow in the NDN-DPDK forwarder, having multiple RX queues for the same face can alleviate FwInput bottleneck.

UDP and VXLAN tunnels can coexist with Ethernet faces on the same port.
Multiple UDP and VXLAN tunnels can coexist if any of the following is true:

* One of *vlan*, *localIP*, and *remoteIP* is different.
* Both are UDP tunnels, and one of *localUDP* and *remoteUDP* is different.
* Between a UDP tunnel and a VXLAN tunnel, the UDP tunnel's *localUDP* is not 4789.
* Both are VXLAN tunnels, and one of *vxlan*, *innerLocal*, and *innerRemote* is different.

Known limitations:

* NDN-DPDK does not respond to Address Resolution Protocol (ARP) or Neighbor Discovery Protocol (NDP) queries.

  * To allow incoming packets to reach NDN-DPDK, configure MAC-IP binding on the IP router.

    ```bash
    sudo ip neigh replace 192.0.2.1 lladdr 5e:c8:55:7a:c9:1f nud noarp dev eth1
    sudo ip neigh replace 2001:0db8::3cfe lladdr 5e:c8:55:7a:c9:1f nud noarp dev eth1
    ```

  * For mlx5 or af\_xdp driver and IPv4: add the IP address to the kernel using `ip addr` command, but do not create the VXLAN interface.
    Even if DPDK is controlling the Ethernet adapter, the kernel can still receive broadcast frames such as ARP queries and respond to them.
    In this case, it is unnecessary to configure MAC-IP binding on the IP router.

* NDN-DPDK does not lookup IP routing tables or send ARP queries.
  To allow outgoing packets to reach the IP router, the *remote* field of the locator should be the MAC address of the IP router.

* IPv4 options and IPv6 extension headers are not allowed.
  Incoming packets with these are dropped.

* IPv4 fragments are not accepted.

* If a VXLAN face has multiple RX queues, NDNLPv2 reassembly works only if all fragments of a network layer packets are sent with the same UDP source port number.
  NDN-DPDK send path and the VXLAN driver in the Linux kernel both fulfill this requirement.

* The default eBPF program used with AF\_XDP driver only supports UDP tunnels on port 6363.
  It does not support UDP tunnels on other ports or VXLAN tunnels.

## Memif Face

Shared memory packet interface (memif) is supported through this package.

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

In the data plane:

* NDN-DPDK and application should operate its memif interface in opposite roles.
* Each packet is an NDN packet without Ethernet header.
