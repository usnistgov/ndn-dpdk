# ndn-dpdk/core

C-only shared code in [csrc/core](../csrc/core/):

* PCG random number generator.
* SipHash hash function.
* uthash hash table library.
* C logging library.

Go shared code:

* cptr: handle C `void*` pointers.
* dlopen: load dynamic libraries.
* logger: Go logging library.
* nnduration: JSON-compatible non-negative duration types.
* runningstat: compute min, max, mean, and variance.
* testenv: unit testing environment.
* urcu: userspace RCU.
* yamlflag: command line flag that accepts a YAML document.
