# ndn-dpdk/core

C shared code in [csrc/core](../csrc/core/):

* PCG random number generator.
* SipHash hash function.
* uthash hash table library.
* C logging library.

Go shared code:

* cptr: handle C `void*` pointers.
* dlopen: load dynamic libraries.
* events: simple event emitter.
* gqlserver: GraphQL server.
* logging: Go logging library.
* macaddr: MAC address parsing and classification.
* nnduration: JSON-compatible non-negative duration types.
* runningstat: compute min, max, mean, and variance.
* testenv: unit testing environment.
* urcu: userspace RCU.
