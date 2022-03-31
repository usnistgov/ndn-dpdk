# ndn-dpdk/core

C shared code in [csrc/core](../csrc/core/):

* common includes
* logging macros
* mmap
* [minute scheduler](mintmr.md)
* RTT estimator
* runningstat
* SipHash wrapper

Go shared code:

* cptr: handle C `void*` pointers.
* dlopen: load dynamic libraries.
* events: simple event emitter.
* gqlserver: GraphQL server.
* hwinfo: hardware information gathering.
* jsonhelper: JSON encoding and decoding.
* logging: Go logging library.
* macaddr: MAC address parsing and classification.
* nnduration: JSON-compatible non-negative duration types.
* pciaddr: PCI address parsing.
* rttest: RTT estimator.
* runningstat: compute min, max, mean, and variance.
* subtract: compute struct numerical difference.
* testenv: unit testing environment.
* urcu: userspace RCU.
* version: version information.
