# ndn-dpdk/container/codel_queue

This package implements a packet queue with [CoDel algorithm](https://tools.ietf.org/html/rfc8289).
This implementation differs from a standard implementation that it dequeues packets in bursts.
The last packet in a burst is used to calculate the sojourn time, and at most one packet can be dropped in each burst.

Acknowledgement: `CoDel_*` functions are adapted from CoDel implementation in Linux kernel, used under BSD license in `codel.LICENSE`.
