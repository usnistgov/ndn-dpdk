# ndn-dpdk/iface/ethport

This package implements faces using DPDK ethdev as transport.

**Face** type represents an Ethernet-based face or a memif face.
See [package ethface](../ethface) and [package memifface](../memifface) for more information.

**Port** type organizes faces on the same DPDK ethdev.
It manages ethdev resources and prevents conflicts among the faces.

**Gtpip** type is a GTP-IP handler.
It contains a hashtable of active GTP-U faces, mapping from UE IP address to FaceID.
It can be attached to a port for forwarding non-NDN traffic in GTP-U tunnels, which enables NDN-DPDK to behave as a 5G User Plane Function (UPF).

## Receive Path

There are three receive path implementations:

* RxFlow: hardware-accelerated receive path.
* RxTable: software receive path.
* RxMemif: memif-specific receive path.

One of them is chosen during port creation.
The choice cannot be changed afterwards.

Each receive path implementation is responsible for:

1. Setup the Ethernet adapter to receive packets toward the faces.
2. Receive packets from the Ethernet adapter.
3. Filter out irrelevant packets, if needed.
4. Label each packet with FaceID and timestamp, and strip packet headers leaving only the NDN packet.

### RxFlow

RxFlow is a hardware-accelerated receive path.
It uses one or more RX queues per face, and creates a *flow* via rte\_flow API to steer incoming frames to those queues (implemented in `C.EthFlowDef` struct).
Some locator schemes have several *variants* implemented to adapt to varying hardware capabilities.
The first flow definition that passes `rte_flow_validate` checks is used for flow creation.

As instructed by the flow definition, the hardware performs packet header matching.
For each matched packet, the hardware sets FaceID as the *mark* value on the mbuf and passes the packet to an RX queue.
The flow isolation mode, if available, is requested to block non-matching packets.

Depending on hardware capability, the software performs minimal checking:

* If the hardware has set the *mark*, the software only checks the mark.
* If the hardware cannot set the *mark*, the software has to perform full header matching.

### RxTable

RxTable is a software receive path.
It continuously polls ethdev RX queue 0 for incoming frames.

For each incoming frame, the software performs header matching (implemented in `C.EthRxMatch` struct), and then labels each matched frame with the face ID.
Matchings are attempted iteratively for each face that are arranged in an RCU-protected linked list.
If the port has a pass-through face, it is arranged last and would always match.
In case no match is found for an incoming frame, the Ethernet frame is sent to [packet dumper](../../app/pdump) if enabled, otherwise it is dropped.

On a port using PCI driver, RxTable opportunistically creates a *flow* via rte\_flow API, which instructs the hardware to set FaceID as the *mark* value.
Upon detecting the *mark*, RxTable bypasses the iterative search and only performs header matching on the indicated face.

On a port using XDP driver, the BPF program can overwrite the Ethernet header of an incoming packet with `C.EthXdpHdr` struct that contains a magic number and the FaceID.
The magic number is UINT64\_MAX, which cannot appear as the first 8 octets of a normal Ethernet header.
Upon detecting this magic number, RxTable bypasses the iterative search and only performs minimal checks of the header length.

### RxMemif

RxMemif is a memif-specific receive path, where each port has only one face.
It continuously polls ethdev RX queue 0 for incoming frames, and then labels each frame with the only face ID.

## Send Path

`EthFace_TxBurst` function implements the send path.
Currently, the send path only uses ethdev TX queue 0.
It prepends Ethernet/UDP/VXLAN headers to each frame (implemented in `EthTxHdr` struct), and requires every outgoing packet to have sufficient headroom for the headers.

The send path is thread-safe only if the underlying DPDK PMD is thread safe, which generally is not the case.
Therefore, **iface.TxLoop** calls `EthFace_TxBurst` from the same thread for all faces on the same port.

## EthLocator Implementation Details

Package ethport supports multiple face schemes, such as Ethernet, UDP, VXLAN, and GTP-U.
Each protocol scheme involves different packet headers, checksum requirements, and hardware filters.
These differences are abstracted in `C.EthLocator` and related types.

Each locator type in package ethface implements `ethport.Locator` face.
It contains an `EthLocatorC` method that populates and returns an `ethport.LocatorC`.

`ethport.LocatorC` is an alias of `C.EthLocator`.
It contains protocol header fields, such as MAC addresses, IP addresses, port numbers, and tunnel identifiers.
From there, it can be converted to four structures: `C.EthRxMatch`, `C.EthXdpLocator`, `C.EthFlowDef`, `C.EthTxHdr`.

`C.EthRxMatch` is used in receive path to determine whether an incoming Ethernet frame is intended for the face.
It is used in **RxTable** and in **RxFlow** when not flow isolated.
It also indicates the header length that should be removed by the receive path implementation.

`C.EthXdpLocator` is used in receive path when an Ethernet device is using AF\_XDP driver.
It is stored in a BPF map that is queried by the XDP program to find the matching face.
Notably, it does not support pass-through face.

`C.EthFlowDef` is used during **RxFlow** setup.
It describes the hardware filters for matching Ethernet frames intended for the face.
Notably, for a GTP-U face, it only checks outer headers up to the GTPv1 header, but ignores inner IPv4+UDP headers.

`C.EthTxHdr` is used in send path to generate protocol headers.
It has a semi-complete buffer of protocol headers that is prepended before each NDNLPv2 packet.
It also comes with a function pointer for the final touches, such as updating length and checksum fields.

## Pass-through Face Implementation Details

This section describes how the pass-through face is implemented.
See [package ethface](../ethface/README.md) for how to use it.

During face creation:

1. `ethport.NewFace` is invoked with an `ethface.PassthruLocator`.

2. As part of `iface.NewParams.Init`, `ethport.passthruInit` overwrites two function pointers on `iface.InitResult`:

    * `initResult.RxInput` is set to `C.EthPassthru_FaceRxInput`.
    * `initResult.TxLoop` is set to `C.EthPassthru_TxLoop`.

3. As part of `iface.NewParams.Start`, `ethport.passthruStart` creates the TAP device.

    * Data structures related to the TAP device is stored in the `C.EthFacePriv` area of the pass-through face.
    * The TAP device is itself an DPDK ethdev and it's activated as an RxGroup here.
    * If GTP-IP handler is enabled, it is created and associated with the pass-through face.

4. As part of `iface.NewParams.Start`, `ethport.rxImpl.Start` is invoked.

    * In RxTable: a pass-through face is appended at the tail of `C.EthRxTable.head` linked list, while faces with other schemes are prepended at the head of this linked list.
    * In RxFlow: a pass-through face is matched with a *flow* with lower priority.

During face teardown:

1. As part of `iface.NewParams.Stop`, `ethport.passthruStop` destroys the TAP device.

    * If GTP-IP handler is enabled, it is destroyed.

Receive path from DPDK ethdev to TAP netif:

1. The receive path identifies which Ethernet frames belong to the pass-through face.

    * In RxFlow: the pass-through face has its own dedicated queue.
    * In RxTable: the pass-through face, located at the tail of `C.EthRxTable.head` linked list, has a `C.EthRxMatch` that matches all packets, so that it will always accept the packet if no other face has accepted it.
    * In RxTable: if the opportunistic *flow* identifies a GTP-U face but the inner IPv4+UDP header mismatches, the packet is dispatched to the pass-through face instead.
      The *mark* value is saved in the mbuf, which could be reused by GTP-IP handler to avoid table lookup.

2. The packet is passed to `C.EthPassthru_FaceRxInput`, which immediately transmits the packet on the TAP netif.

    * If GTP-IP handler is enabled, `C.EthGtpip_ProcessUplink` is invoked.
      If the packet is recognized as GTP-U and matches an existing GTP-U tunnel face, the packet is modified for outer header removal.
    * These operations occur in the RX thread.
      The packet does not go through the TX thread.

Send path from TAP netif to DPDK ethdev:

1. The RxGroup for the TAP device uses `C.EthPassthru_TapPortRxBurst` as its `C.RxGroup_RxBurstFunc` function pointer.

    * `C.EthPassthru_TapPortRxBurst` receives a burst of Ethernet frames from the TAP netif and immediately enqueues them for transmission on the pass-through face.
    * This occurs in the RX thread.

2. The TxLoop of the pass-through face uses `C.EthPassthru_TxLoop` as its `C.Face_TxLoopFunc` function pointer.

    * `C.EthPassthru_TxLoop` dequeues outgoing packets for the face.
    * If GTP-IP handler is enabled, `C.EthGtpip_ProcessDownlinkBulk` is invoked.
      If the destination IP address matches an existing GTP-U tunnel face, the packet is modified for outer header creation.
    * The packet is then passes to `C.TxLoop_TxFrames` without going through NDNLPv2 fragmentation.
    * These operations occur in the TX thread.

3. Finally, `C.TxLoop_TxFrames` invokes `C.FaceImpl.txBurst` to transmit the packet.

    * `C.FaceImpl.txBurst` is usually `C.EthFace_TxBurst`.
