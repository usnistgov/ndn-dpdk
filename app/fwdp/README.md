# ndn-dpdk/app/fwdp

This package implements the forwarder's data plane.

## Input Thread (FwInput)

An FwInput thread runs an **iface.RxLoop** as its main loop ("RX" role), which reads and decodes packets from one or more network interfaces.
Bursts of received L3 packets are processed by InputDemux, configured to use the [NDT](../../container/ndt) for Interests and the PIT token for Data and Nacks.

## Forwarding Thread (FwFwd)

An FwFwd thread runs the `FwFwd_Run` function as its main loop ("FWD" role).
The main loop first performs some maintenance work:

* Mark a URCU quiescent state, as required by the FIB.
* Trigger the PIT timeout scheduler.

Then it reads packets from the input queues and handles each packet separately:

* `FwFwd_RxInterest` function handles an incoming Interest.
* `FwFwd_RxData` function handles an incoming Data.
* `FwFwd_RxNack` function handles an incoming Nack.

### Data Structure Usage

All FwFwd threads have read-only access to a shared [FIB](../../container/fib) replica on the same NUMA socket.
Each FwFwd thread has read-write access to a `FibEntryDyn` struct associated with each FIB entry.

Each FwFwd has a private partition of [PIT and CS](../../container/pcct).
An outgoing Interest from a FwFwd must carry the identifier of this FwFwd as the first 8 bits of its PIT token, so that returning Data or Nack can be dispatched to the same FwFwd and thus use the same PIT-CS partition.

### Congestion Control

Each FwFwd has three [CoDel queues](../../iface), one for each L3 packet type.
They are backed by DPDK rings in multi-producer single-consumer mode.
An FwInput thread enqueues packets to these queues; in case the DPDK ring is full, the packet is dropped.
An FwFwd dequeues packets from these queues; if the CoDel algorithm indicates a packet should be dropped, FwFwd places a congestion mark on the packet but does not drop it.
The ratio of dequeue burst size among the three queues determines the relative weight among L3 packet types; for example, dequeuing up to 48 Interests, 64 Data, and 64 Nacks would give Data/Nacks priority over Interests.

Note that congestion mark handling is currently incomplete.
Some limitations are:

* FwFwd can place a congestion mark only on the ingress side (e.g., to signal that the forwarder cannot sustain the current rate of incoming packets), not on the egress side (e.g., to signal link congestion).
* FwFwd does not add or remove the congestion mark during Interest aggregation or Data caching.

### Per-Packet Logging

FwFwd C code uses the `DEBUG` log level for per-packet logging.
Generally, a log line has several key-value pairs delimited by whitespace.
Keys use "kebab-case".
Common keys include:

* "interest-from", "data-from", "nack-from": incoming FaceID in packet arrival.
* "interest-to", "data-to", "nack-to": outgoing FaceID in packet transmission.
* "npkt" (meaning "NDN packet"): memory address of a packet.
* "dn-token": PIT token at the downstream node.
* "up-token": PIT token assigned by this node, which is sent upstream.
* "drop": reason for dropping a packet.
* "pit-entry", "cs-entry": memory address of a table entry.
* "pit-key": debug string of a PIT entry.
* "sg-id": strategy identifier.
* "sg-res": return value of a strategy invocation.
* "helper": handing off to a helper.

## Crypto Helper (FwCrypto)

FwCrypto provides implicit digest computation for Data packets.
When an FwFwd processes an incoming Data packet and finds a PIT entry whose Interest carries an `ImplicitSha256DigestComponent`, it needs to know the Data's implicit digest in order to determine whether the Data satisfies the Interest.
Instead of performing the digest computation synchronously, which would block the processing of other packets, the FwFwd passes the Data to FwCrypto.
After the digest is computed, the Data packet goes back to FwFwd, which can then re-process it and use the computed digest to determine whether it satisfies the pending Interest.

An FwCrypto thread runs the `FwCrypto_Run` function as its main loop ("CRYPTO" role).
It receives Data packets from FwFwd threads through a queue, and enqueues crypto operations toward a DPDK cryptodev.
The cryptodev computes the SHA-256 digest of the packet and stores it in the mbuf header.
The FwCrypto then dequeues the completed crypto operations from the cryptodev and re-dispatches the Data to FwFwd in the same fashion as an input thread.

It is possible to disable FwCrypto by assigning zero lcores to "CRYPTO" role.
In this case, the forwarder does not support implicit digest computation, and incoming Interests with implicit digest component are dropped.

## Disk Helper (FwDisk)

FwDisk enables on-disk caching in the Content Store.
See [package disk](../../container/disk) for general concepts.

This implementation is work in progress.
Currently, it can only use emulated block device with Malloc or file backend, but not a hardware NVMe device.
