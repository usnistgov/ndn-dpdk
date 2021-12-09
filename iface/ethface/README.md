# ndn-dpdk/iface/ethface

This package implements Ethernet-based faces using DPDK ethdev as transport.

**ethFace** type represents an Ethernet-based face.
This includes NDN over Ethernet (with optional VLAN header), UDP tunnel, and VXLAN tunnel.
See [face creation](../../docs/face.md) "creating Ethernet-based face" section for locator syntax.

**Port** type organizes faces on the same DPDK ethdev.
Each port can have zero or one Ethernet face with multicast remote address, and zero or more Ethernet faces with unicast remote addresses.
Faces on the same port can be created and destroyed individually.

## Receive Path

There are three receive path implementations.
One of them is chosen during port creation; the choice cannot be changed afterwards.

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

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev TX queue 0.
It requires every outgoing packet to have sufficient headroom for the Ethernet header.

The send path is thread-safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Therefore, **iface.TxLoop** calls `EthFace_TxBurst` from the same thread for all faces on the same port.

## UDP and VXLAN Tunnel Face

UDP and VXLAN tunnels are supported through this package.

UDP and VXLAN tunnels can coexist with Ethernet faces on the same port.
Multiple UDP and VXLAN tunnels can coexist if any of the following is true:

* One of *vlan*, *localIP*, and *remoteIP* is different.
* Both are UDP tunnels, and one of *localUDP* and *remoteUDP* is different.
* Between a UDP tunnel and a VXLAN tunnel, the UDP tunnel's *localUDP* is not 4789.
* Both are VXLAN tunnels, and one of *vxlan*, *innerLocal*, and *innerRemote* is different.

Caveats and limitations:

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
See [face creation](../../docs/face.md) "memif face" section for locator syntax.

In the data plane:

* NDN-DPDK and application should operate its memif interface in opposite roles.
* Each packet is an NDN packet without Ethernet header.
