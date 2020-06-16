# ndn-dpdk/spdk

This directory contains Go bindings for [Storage Performance Development Kit (SPDK)](https://spdk.io/).

## Go bindings

* thread
* poller
* bdev

Go bindings are object-oriented when possible.

## SPDK Threads

Many SPDK library functions must run on an SPDK thread.
`spdkenv` package creates and launches a `MainThread` on a DPDK lcore.
Most operations invoked from Go API are executed on this thread.

## Internal RPC Client

Several SPDK features are not exposed in SPDK headers, but only accessible by its [JSON-RPC server](https://spdk.io/doc/jsonrpc.html).
`spdkenv` package enables SPDK's JSON-RPC server, and creates an internal JSON-RPC client to send commands to this server.
