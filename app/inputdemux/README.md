# ndn-dpdk/app/inputdemux

This package provides an input packet demultiplexer.

The **InputDemux** type can dispatch packets according to one of these criteria:

* Drop all packets.
* Always use the first destination.
* Cycle through several destinations in a round-robin fashion.
* Choose the destination by querying the [NDT](../../container/ndt) with the packet name.
* Use the 8 most-significant bits of the PIT token to determine the destination.

InputDemux also provides counters that record how many packets were queued toward each destination and how many were dropped due to full queues.

The **InputDemux3** type aggregates three InputDemux instances, one for each network-layer packet type.
The `InputDemux3_FaceRx` function can be used with [iface.RxLoop](../../iface) to process bursts of ingress packets.
