# ndn-dpdk/core/urcu

This package wraps the [Userspace RCU](http://liburcu.org/) library.

NDN-DPDK uses the QSBR flavor of RCU (liburcu-qsbr).
Therefore, every read-side thread must execute `func (*ReadSide) Quiescent()` (in Go) or `rcu_quiescent_state()` (in C) in its main loop to ensure progress.
