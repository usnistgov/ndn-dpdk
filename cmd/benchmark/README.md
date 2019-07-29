# ndn-dpdk/cmd/benchmark

This directory contains forwarder benchmark tools.
These tools expect an [`ndnping-dpdk`](../ndnping-dpdk/) traffic generator with one or more clients, and control the generator via [JSON-RPC](../../mgmt/pingmgmt/).

`msi.ts` attempts to find **minimum sustained interval** (MSI).
It minimizes the Interest sending interval, such that Interest satisfaction ratio stays near 100% within a period of time.
Measured MSI can be used to calculate the throughput of a forwarder or a network.
