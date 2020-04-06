# ndn-dpdk/app/inputdemux

This package provides an input packet demultiplexer.

**InputDemux** type can dispatch packets using one of these methods:

* Drop all packets.
* Round-robin among several destinations.
* Alway use first (and only) destination.
* Choose destination by querying [NDT](../../container/ndt) with name.
* Use high 8 bits of PIT token as destination.

It has counters that record how many packets have been queued to each destination, and how many are dropped due to full queues.

**InputDemux3** type aggregates InputDemux instances for all three network layer packet types.
The `InputDemux3_FaceRx` function can be used with [iface.RxLoop](../../iface) to process bursts of ingress packets.
