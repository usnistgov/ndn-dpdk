# ndn-dpdk/dpdk

This directory contains Go bindings for the [Data Plane Development Kit (DPDK)](https://www.dpdk.org/).

## C extensions

The following C extensions are provided in [csrc/dpdk](../csrc/dpdk):

* `MbufLoc`: iterator in a multi-segment mbuf.

## Go bindings

* EAL, lcore, launch
* mempool, mbuf
* ring
* ethdev
* cryptodev

Go bindings are object-oriented when possible.

## Other Go types

**ealthread.Thread** abstracts a thread that can be executed on an LCore and controls its lifetime.

**ealthread.Allocator** provides an LCore allocator.
It allows a program to reserve a number of LCores for each "role", and then obtain a NUMA-local LCore reserved for a certain role when needed.

**pktmbuf.Template** is a template of mempool configuration.
It can be used to create per-NUMA mempools for packet buffers.
