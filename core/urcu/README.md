# ndn-dpdk/core/urcu

This package provides a wrapper of [Userspace RCU](http://liburcu.org/).

The QSBR flavor of RCU (liburcu-qsbr) is being used. Therefore, every read-side thread must execute `func (*ReadSide) Quiescent()` (in Go) or `rcu_quiescent_state()` (in C) in its main loop to ensure progress.
