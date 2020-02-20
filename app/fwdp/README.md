# ndn-dpdk/app/fwdp

This package implements the forwarder's data plane.

The data plane consists two types of threads, *input thread* and *forwarding thread*.
Each thread runs in a DPDK lcore, allocated from "RX" or "FWD" role.

## Input Thread (FwInput)

A FwInput runs an **iface.RxLoop** as the main loop ("RX" role), which reads and decodes packets from one or more network interfaces.
Bursts of received L3 packets are processed by [InputDemux3](../inputdemux), configured to use NDT for Interests, and high 8 bits for Data and Nacks.

## Crypto Helper (FwCrypto)

FwCrypto provides Data implicit digest computation.
It runs `FwCrypto_Run` as the main loop ("CRYPTO" role).

When FwFwd threads an incoming Data packet and finds a PIT entry whose Interest carries the ImplicitSha256DigestComponent, it needs to compute the Data's implicit digest in order to determine whether the Data satisfies the Interest.
Instead of doing the computation in FwFwd and blocking other packet processing, the FwFwd passes the Data to FwCrypto.
FwCrypto computes Data digest using a DPDK cryptodev, stores the implicit digest in the mbuf header, and re-dispatches the Data to FwFwd using [InputDemux](../inputdemux).
FwFwd can then re-process the Data, and use the computed implicit digest to determine whether it satisfies the pending Interest.

## Forwarding Thread (FwFwd)

A FwFwd runs `FwFwd_Run` function as the main loop ("FWD" role).
The main loop first performs some maintenance work:

* Mark a URCU quiescent state, as required by FIB.
* Trigger the PIT timeout scheduler.

Then it reads packets from input queues, and handles each packet separately:

* `FwFwd_RxInterest` function handles an incoming Interest.
* `FwFwd_RxData` function handles an incoming Data.
* `FwFwd_RxNack` function handles an incoming Nack.

Each FwFwd has three [CoDel queues](../../container/pktqueue/), one for each L3 packet type.
They are backed by DPDK rings in multi-producer single-consumer mode.
FwInputs enqueue packets to these queues; in case the DPDK ring is full, FwInput drops the packet.
FwFwds dequeue packets from these queues; if CoDel algorithms indicates a packet could be dropped, FwFwd places a congestion mark on the packet but does not drop the packet.
The ratio of dequeue burst size among the three queues determines relative weight among L3 packet types; for example, dequeuing up to 48 Interests, 64 Data, and 64 Nacks would give Data/Nack a priority over Interest.

Congestion mark handling is incomplete.
Some limitations are:

* FwFwd can place congestion mark only on ingress side (i.e. insufficient processing power), not on egress side (i.e. link congestion).
* FwFwd does not add or remove congestion mark during Interest aggregation or Data caching.
* FwFwd does not place congestion mark on reply Data/Nack when Interest congestion occurs, although the producer could do so.

### Data Structure Usage

All FwFwds have read-only access to a shared [FIB](../../container/fib/).

Each FwFwd has a private partition of [PIT-CS](../../container/pcct/).
An outgoing Interest from a FwFwd must carry the identifier of this FwFwd as the first 8 bits of its PIT token, so that returned Data or Nack can come back to the same FwFwd and thus use the same PIT-CS partition.

### Per-Packet Logging

`FwFwd` uses DEBUG log level for per-packet logging.
Generally, a log line has several key-value pairs delimited by space.
Keys should use kebab-case.
Common keys include:

* "interest-from", "data-from", and "nack-from": incoming FaceId in packet arrival.
* "interest-to", "data-to", or "nack-to": outgoing FaceId in packet transmission.
* "npkt" (meaning "NDN packet"): memory address of a packet.
* "dn-token": PIT token at downstream.
* "up-token": PIT token assigned by this node, which is sent upstream.
* "drop": reason of dropping a packet.
* "pit-entry" and "cs-entry": memory address of a table entry.
* "pit-key": debug string of a PIT entry.
* "sg-id": strategy identifier.
* "sg-res": return value of strategy invocation.
* "helper": handing off to a helper.
