# ndn-dpdk/dpdk

This package contains Go bindings for DPDK, as well as extensions to DPDK in C.

Unit tests of this package are in [dpdktest](dpdktest/) subdirectory.

## C extensions

* `MbufLoc`: iterator in a multi-segment mbuf

## Go bindings

This package has Go bindings for:

* EAL, lcore, launch
* mempool, mbuf
* ring
* ethdev
* cryptodev

Go bindings are object-oriented when possible.
