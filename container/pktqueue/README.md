# ndn-dpdk/container/pktqueue

This package implements a packet queue that can operate in one of three modes.

**Plain** mode: simple drop-tail queue.

**Delay** mode: drop-tail queue, enforcing a minimum amount of delay.
This is useful for simulating a processing delay.

**CoDel** mode: queue with [CoDel algorithm](https://tools.ietf.org/html/rfc8289).
This CoDel implementation differs from a standard implementation that it dequeues packets in bursts.
The last packet in a burst is used to calculate the sojourn time, and at most one packet can be dropped in each burst.
`CoDel_*` functions are adapted from CoDel implementation in Linux kernel, used under BSD license in `codel.LICENSE`.
