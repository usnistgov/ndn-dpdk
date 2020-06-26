# ndn-dpdk/container/pktqueue

This package implements a packet queue that can operate in one of three modes.

**Plain** mode: a simple drop-tail queue.

**Delay** mode: a drop-tail queue that enforces a minimum amount of delay.
This is useful for simulating a processing delay.

**CoDel** mode: a queue that uses the [CoDel algorithm](https://tools.ietf.org/html/rfc8289).
This CoDel implementation differs from a standard implementation in that it dequeues packets in bursts instead of one at a time.
The last packet in each burst is used to calculate the sojourn time, and at most one packet can be dropped in each burst.
The `CoDel_*` functions are adapted from the CoDel implementation in the Linux kernel, under the BSD license (see [`codel.LICENSE`](codel.LICENSE)).
