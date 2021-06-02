# ndn-dpdk/dpdk

This directory contains Go bindings for the [Data Plane Development Kit (DPDK)](https://www.dpdk.org/) and the [Storage Performance Development Kit (SPDK)](https://spdk.io/).

## Go Bindings

DPDK:

* EAL, lcore, launch
* mempool, pktmbuf
* ring
* vdev
* ethdev
* cryptodev

SPDK:

* thread
* poller
* bdev

Go bindings are object-oriented when possible.

## Other Go Types and Functions

**ealthread.Thread** abstracts a thread that can be executed on an LCore and controls its lifetime.

**ealthread.Allocator** provides an LCore allocator.
It allows a program to reserve a number of LCores for each "role", and then obtain a NUMA-local LCore reserved for a certain role when needed.

**ealthread.LoadStat** collects statistics of thread empty polls.
It is a simplified version of DPDK empty poll API, intended for informational purpose instead of power management.

**pktmbuf.Template** is a template of mempool configuration.
It can be used to create per-NUMA mempools for packet buffers.

**ethringdev.Pair** builds a pair of virtual Ethernet devices based on rte\_net\_ring driver.
**ethringdev.VNet** further builds a virtual network of bridged Ethernet devices.
They can be used in unit tests to simulate links.

**ethvdev** has functions to create virtual Ethernet devices based on rte\_net\_memif, rte\_net\_af\_xdp, and rte\_net\_af\_packet drivers.

## Main Thread

Certain DPDK library functions must run on the initial lcore; certain SPDK library functions must run on an SPDK thread.
To satisfy both requirements, package `ealinit` creates and launches a main thread on the initial lcore.
This thread is initialized as an SPDK thread, and is also registered as a [URCU](../core/urcu) read-side thread.
Most operations invoked via the Go API are executed on this thread.

## SPDK Internal RPC Client

Several SPDK features are not exposed in SPDK header files, but are only accessible through its [JSON-RPC server](https://spdk.io/doc/jsonrpc.html).
Package `spdkenv` enables SPDK's JSON-RPC server and creates an internal JSON-RPC client to send commands to it.
