# ndn-dpdk/app/fwdp

This package implements the data plane of a forwarder.

The data plane consists two types of processes, "input process" and "forwarding process".
Each process runs in a DPDK lcore.

## Input Process (FwInput)

A FwInput runs an **iface.RxLooper** as the main loop, which reads and decodes packets from one or more network interfaces.
Every burst of receives L3 packets triggers `FwInput_FaceRx` function.

For each incoming packet, FwInput decides which forwarding process should handle the packet:

* For an Interest, lookup the [NDT](../../container/ndt/) with the Interest name.
* For a Data or Nack, take the first 8 bits of its PIT token.

Then, FwInput passes the packet to the chosen forwarding process's input queue (a DPDK ring in multi-produer single-consumer mode).
In case the queue is full, FwInput drops the packet, and increments a drop counter.

### Data Structure Usage

All FwInputs have read-only access to a shared NDT.

## Forwarding Process (FwFwd)

A FwFwd runs `FwFwd_Run` function as the main loop.
It reads packets from its input queue, and handles each packet separately:

* `FwFwd_RxInterest` function handles an incoming Interest.
* `FwFwd_RxData` function handles an incoming Data.
* `FwFwd_RxNack` function handles an incoming Nack.

Before reading each burst of input packets, the main loop also performs some maintenance work:

* Mark a URCU quiescent state, as required by FIB.
* Trigger the PIT timeout scheduler.

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
