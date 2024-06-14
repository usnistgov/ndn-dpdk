# ndn-dpdk/iface/ethface

This package implements Ethernet-based faces using DPDK ethdev as transport.
This includes Ethernet faces (with optional VLAN header), UDP faces, and VXLAN faces.
Additional, GTP-U faces feature is in development.
See [face creation](../../docs/face.md) "creating Ethernet-based face" section for locator syntax.

The underlying implementation is in [package ethport](../ethport).

## Ethernet Face

Each port can have zero or one Ethernet face with multicast remote address, and zero or more Ethernet faces with unicast remote addresses.
Faces on the same port can be created and destroyed individually.

Caveats and limitations:

* It's possible to set a local MAC address that differs from the hardware MAC address.
  However, this may not work properly on certain hardware, and thus is not recommended.

* It's possible to create both VLAN-tagged faces and faces without VLAN headers.
  However, this may not work properly on certain hardware, and thus is not recommended.

## UDP and VXLAN Tunnel Face

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

  * For mlx5 or XDP driver and IPv4: add the IP address to the kernel using `ip addr` command, but do not create the VXLAN interface.
    Even if DPDK is controlling the Ethernet adapter, the kernel can still receive broadcast frames such as ARP queries and respond to them.
    In this case, it is unnecessary to configure MAC-IP binding on the IP router.

* NDN-DPDK does not lookup IP routing tables or send ARP queries.
  To allow outgoing packets to reach the IP router, the *remote* field of the locator should be the MAC address of the IP router.

* IPv4 options and IPv6 extension headers are not allowed.
  Incoming packets with these are dropped.

* IPv4 fragments are not accepted.

* If a VXLAN face has multiple RX queues, NDNLPv2 reassembly works only if all fragments of a network layer packets are sent with the same UDP source port number.
  NDN-DPDK send path and the VXLAN driver in the Linux kernel both fulfill this requirement.
