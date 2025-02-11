# ndn-dpdk/iface/ethface

This package implements Ethernet-based faces using DPDK ethdev as transport.
This includes Ethernet faces (with optional VLAN header), UDP faces, VXLAN faces, and GTP-U faces.
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

## UDP, VXLAN, GTP-U Tunnel Face

UDP, VXLAN, GTP-U tunnels can coexist with Ethernet faces on the same port.
Multiple tunnels can coexist if any of the following is true:

* One of *vlan*, *localIP*, and *remoteIP* is different.
* Both are UDP tunnels, and one of *localUDP* and *remoteUDP* is different.
* Between a UDP tunnel and a VXLAN tunnel, the UDP tunnel's *localUDP* is not 4789.
* Between a UDP tunnel and a GTP-U tunnel, the UDP tunnel's *localUDP* is not 2152.
* Both are VXLAN tunnels, and one of *vxlan*, *innerLocal*, and *innerRemote* is different.
* Both are GTP-U tunnels, and both *ulTEID* and *dlTEID* are different.

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

  * You can also create a pass-through face and assign the IP address on the associated TAP network interface.

* NDN-DPDK does not lookup IP routing tables or send ARP queries.
  To allow outgoing packets to reach the IP router, the *remote* field of the locator should be the MAC address of the IP router.

* IPv4 options and IPv6 extension headers are not allowed.
  Incoming packets with these are dropped.

* IPv4 fragments are not accepted.

* If a VXLAN face has multiple RX queues, NDNLPv2 reassembly works only if all fragments of a network layer packet are sent with the same UDP source port number.
  NDN-DPDK send path and the VXLAN driver in the Linux kernel both satisfy this requirement.

### More on GTP-U Tunnel Face

The GTP-U tunnel face is designed to work with IPv4 session type.
The overall packet structure is as follows:

1. Outer Ethernet header.
2. Outer VLAN header (optional).
3. Outer IPv4 or IPv6 header.
4. Outer UDP header.
5. GTPv1 header.
6. Inner IPv4 header.
7. Inner UDP header.
8. NDNLPv2 packet.

## Pass-through Face

The pass-through face allows receiving and sending non-NDN traffic, on a DPDK ethdev exclusively occupied by NDN-DPDK.
To create a pass-through face, using the locator:

```jsonc
{
  "scheme": "passthru",
  "port": "b9RENroce85E",      // ethdev GraphQL ID, or
  "local": "02:00:00:00:00:00" // ethdev local MAC address
}
```

Each port can have at most one pass-through face.
The pass-through face is associated with a TAP netif, which has the same local MAC address and MTU as the ethdev.
Packets sent to the TAP netif are transmitted out of the ethdev.
Packets received by the ethdev that do not match an NDN face are received by the TAP netif.

Caveats and limitations:

* Currently, this only works with RxTable.
* TAP netif name is unchangeable.
* In packet counters, all packets are considered "Interests".
* This is incompatible with [packet dumper](../../app/pdump).

See [package ethport](../ethport/README.md) for implementation details.
