# NDN-DPDK Interactive Benchmark

This is a web application that performs simple benchmark of NDN-DPDK forwarder.

## Instructions

### Hardware Requirements

* Two physical machines.
* Two PCI Ethernet adapters, connected via direct attach cables.
  * Both adapters must be on the same NUMA socket.
  * The adapter must support PCI driver and RxFlow with 2 queues.
  * The link must support MTU 9000.
* At least 8 CPU cores on the same NUMA socket as the Ethernet adapters.
* At least 2 CPU cores on any NUMA socket.

### NDN-DPDK Setup

1. Install NDN-DPDK as systemd service.
   * You should compile NDN-DPDK in release mode, see [installation guide](../INSTALL.md) "compile-time settings" section.
2. Setup CPU isolation, see [performance tuning](../tuning.md) "CPU isolation" section.
3. Follow through [forwarder activation](../forwarder.md) "ndnping" scenario to ensure the forwarder works.

### Usage

1. Make a copy of this directory to somewhere outside the NDN-DPDK repository.
2. Run `npm install` to install dependencies.
3. Copy `sample.env` as `.env`, and then edit `.env` according to the comments within.
4. Start NDN-DPDK service on both machines.
5. Start SSH tunnel for reaching remote NDN-DPDK.
6. Run `npm start` to start the web application.
7. Visit `http://localhost:3333` (via SSH tunnel) in your browser.

## Benchmark Description

This benchmark controls a forwarder and a traffic generator via GraphQL.

The forwarder is setup as follows:

* There are *n* forwarding threads.
* Face A has FIB prefixes `/A/0`, `/A/1`, &hellip;, `/A/`*n-1*.
* Face B has FIB prefixes `/B/0`, `/B/1`, &hellip;, `/B/`*n-1*.
* NDT is setup so that traffic is balanced: `/A/`*i* and `/B/`*i* prefixes are dispatched to forwarding thread #*i*.
* Caching is minimal and cache hits are not expected.

The traffic generator is setup as follows:

* Face A has producer A serving infinitely large "files" under `/A/0`, `/A/1`, &hellip;, `/A/`*n-1* prefixes.
* Face B has producer B serving infinitely large "files" under `/B/0`, `/B/1`, &hellip;, `/B/`*n-1* prefixes.
* Face A has consumers fetching from producer B through the forwarder.
* Face B has consumers fetching from producer A through the forwarder.
* An Interest name looks like `/A/0/i/i/i/seg=202`, in which `/i` is repeated to make up desired name length.
* If Data "prefix match" is selected, Data name is Interest name plus `/d` suffix; otherwise, Data name is same as Interest name.

[benchmark.ts](src/benchmark.ts) implements the core benchmark logic.
Read the code to understand the exact parameters, or use it as a starting point for developing other benchmarks.
