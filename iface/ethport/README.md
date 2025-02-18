# ndn-dpdk/iface/ethport

This package implements faces using DPDK ethdev as transport.

**Face** type represents an Ethernet-based face or a memif face.
See [package ethface](../ethface) and [package memifface](../memifface) for more information.

**Port** type organizes faces on the same DPDK ethdev.
It manages ethdev resources and prevents conflicts among the faces.

## Receive Path

There are three receive path implementations.
One of them is chosen during port creation; the choice cannot be changed afterwards.

**RxFlow** is a hardware-accelerated receive path.
It uses one or more RX queues per face, and creates a *flow* via rte\_flow API to steer incoming frames to those queues.
The hardware performs header matching; there is minimal checking on software side.

**RxTable** is a software receive path.
It continuously polls ethdev RX queue 0 for incoming frames.
For each incoming frame, the software performs header matching (implemented in `EthRxMatch` struct), and then labels each matched frame with the face ID.
Matchings are attempted iteratively for each face that are arranged in an RCU-protected linked list; if the port has a pass-through face, it is arranged last and would always match.
If no match is found for an incoming frame, the Ethernet frame is sent to [packet dumper](../../app/pdump) if enabled, otherwise it is dropped.

**RxMemif** is a memif-specific receive path, where each port has only one face.
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
From there, it can be converted to four structures: `C.EthRxMatch`, `C.EthXdpLocator`, `C.EthFlowPattern`, `C.EthTxHdr`.

`C.EthRxMatch` is used in receive path to determine whether an incoming Ethernet frame is intended for the face.
It is used in **RxTable** and in **RxFlow** when not flow isolated.
It also indicates the header length that should be removed by the receive path implementation.

`C.EthXdpLocator` is used in receive path when an Ethernet device is using AF\_XDP driver.
It is stored in a BPF map that is queried by the XDP program to find the matching face.

`C.EthFlowPattern` is used during **RxFlow** setup.
It describes the hardware filters for matching Ethernet frames intended for the face.

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

    * Handling of pass-through face is only implemented in `ethport.rxTable`.
    * A pass-through face is appended at the tail of `C.EthRxTable.head` linked list, while faces with other schemes are prepended at the head of this linked list.

During face teardown:

1. As part of `iface.NewParams.Stop`, `ethport.passthruStop` destroys the TAP device.

    * If GTP-IP handler is enabled, it is destroyed.

Receive path from DPDK ethdev to TAP netif:

1. `C.EthRxTable_RxBurst` receives a burst of Ethernet frames and calls `C.EthRxTable_Accept` on each packet to find which faces could accept it.

2. The pass-through face is at the tail of `C.EthRxTable.head` linked list and has an `C.EthRxMatch` that matches all packets, so that it will always accept the packet if no other face has accepted it.

3. The packet is passed to `C.EthPassthru_FaceRxInput`, which immediately transmits the packet on the TAP netif.

    * If GTP-IP handler is enabled, `C.EthGtpip_ProcessUplink` is invoked.
      If the packet is recognized as GTP-U and matches an existing GTP-U tunnel face, the packet is modified with outer header removal.
    * These operations occur in the RX thread.
      The packet does not go through the TX thread.

Send path from TAP netif to DPDK ethdev:

1. The RxGroup for the TAP device uses `C.EthPassthru_TapPortRxBurst` as its `C.RxGroup_RxBurstFunc` function pointer.

    * `C.EthPassthru_TapPortRxBurst` receives a burst of Ethernet frames from the TAP netif and immediately enqueues them for transmission on the pass-through face.
    * This occurs in the RX thread.

2. The TxLoop of the pass-through face uses `C.EthPassthru_TxLoop` as its `C.Face_TxLoopFunc` function pointer.

    * `C.EthPassthru_TxLoop` dequeues outgoing packets for the face.
    * If GTP-IP handler is enabled, `C.EthGtpip_ProcessDownlink` is invoked.
      If the destination IP address matches an existing GTP-U tunnel face, the packet is modified with outer header creation.
    * The packet is then passes to `C.TxLoop_TxFrames` without going through NDNLPv2 fragmentation.
    * These operations occur in the TX thread.

3. Finally, `C.TxLoop_TxFrames` invokes `C.FaceImpl.txBurst` to transmit the packet.

    * `C.FaceImpl.txBurst` is usually `C.EthFace_TxBurst`.
