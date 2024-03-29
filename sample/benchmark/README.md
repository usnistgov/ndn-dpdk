# NDN-DPDK Interactive Benchmark

This is a web application that performs NDN-DPDK benchmarks.

## Benchmark Description

The benchmark topology consists of three logical nodes, connected linearly:

* traffic generator **A**
* forwarder **F**
* traffic generator **B**

The forwarder is setup as follows:

* There are *n* forwarding threads.
* Face A has FIB prefixes `/A/0`, `/A/1`, &hellip;, `/A/<n-1>`.
* Face B has FIB prefixes `/B/0`, `/B/1`, &hellip;, `/B/<n-1>`.
* NDT is setup so that traffic is balanced: `/A/<i>` and `/B/<i>` prefixes are dispatched to forwarding thread #*i*.
* Caching is minimal and cache hits are not expected.

The traffic generator **A** is setup as follows:

* It has a producer and a consumer, attached to the same face that is connected to the forwarder's face A.
* The producer is either [traffic generator producer](../../app/tgproducer) or [file server](../../app/fileserver).
  * If using a traffic generator producer, it replies to each Interest with fixed-length Data packets, simulating infinite-length "files".
  * If using a file server, it serves a pre-generated file, under several different names via symbolic links.
* The consumer is [congestion aware fetcher](../../app/fetch).
  * It retrieves *n* files from producer B through the forwarder.
  * Each file retrieval stops when the time limit or segment count limit is reached, whichever occurs earlier.
* An Interest name looks like `/A/<i>/<j>/I/I/I/<seg>`, where:
  * *i* is the forwarding thread index, to balance load among forwarding threads.
  * *j* is a random number between 0 and 1023, unique among different flows during a run, to balance load among producer threads.
  * `/I` is repeated to make up desired name length.
  * *seg* is file segment number.
* If Data "prefix match" is selected, Data name is Interest name plus `/D` suffix.
  Otherwise, Data name is same as Interest name.

The traffic generator **B** is setup similarly.
If unidirectional traffic is selected on the webapp, only producer A and consumer B are active.

[benchmark.ts](src/benchmark.ts) implements the core benchmark logic:

1. Restart NDN-DPDK service instances, to clear states from any prior benchmarks.
2. Activate the forwarder and traffic generators.
3. Start the fetchers to retrieve *n* files in parallel.
4. Subscribe to fetch counters, save the counters (1) after warmup period (2) upon retrieval completion or at the end of trial duration.
5. Stop the fetchers.
6. Calculate throughput from the difference between two sets of counters, reporting the average among all flows.
7. Go to step 3.

Notably, the forwarder and producers are not restarted between consecutive trials.
Read the code to understand the exact parameters, or use it as a starting point for developing other benchmarks.

## Hardware Requirements

The benchmark controls either two or three NDN-DPDK service instances via GraphQL:

* The forwarder needs one NDN-DPDK service instance.
* If two traffic generators are on the same host machine, they can run in the same NDN-DPDK service instance.
  * This arrangement saves 1 CPU core.
* Each traffic generator can have its own NDN-DPDK service instance.

If the host machine has multiple NUMA sockets, you must designate a primary NUMA socket.
Most of the CPU cores, as well as physical Ethernet adapters, must be on the primary NUMA socket.

The forwarder needs, at minimum:

* 8 CPU cores on the primary NUMA socket
* 2 CPU cores on any NUMA socket
* 12 GB hugepages

When each traffic generator has its own NDN-DPDK service instance, it needs, at minimum:

* 5 CPU cores on the primary NUMA socket
* 1 CPU core on any NUMA socket
* 8 GB hugepages

When two traffic generators share the same NDN-DPDK service instance, it needs, at minimum:

* 10 CPU cores on the primary NUMA socket
* 1 CPU core on any NUMA socket
* 8 GB hugepages

These are minimal requirements to run 1 forwarding thread.
More resources may be needed to enable more forwarding threads.

Each network connection, A-F and B-F, can use either physical Ethernet adapters or memif virtual interfaces.

If using physical Ethernet adapters:

* Each adapter must support PCI driver and have RxFlow feature with 2 queues.
* Two adapters must connect to each other, either directly or over a VLAN.
* The link must support MTU 9000.
* Each adapter must be on the primary NUMA socket of the owning NDN-DPDK service instance.
  You can determine NUMA socket by looking at `/sys/bus/pci/devices/*/numa_node` file.

If using memif virtual interfaces:

* Two NDN-DPDK service instances must run on the same host machine and have access to the same `/run/ndn` directory.

## Instructions

### NDN-DPDK Setup

1. Install NDN-DPDK as systemd service.
   * You should compile NDN-DPDK in release mode, see [installation guide](../../docs/INSTALL.md) "compile-time settings" section.
2. Setup CPU isolation, see [performance tuning](../../docs/tuning.md) "CPU isolation" section.
3. Follow through [forwarder activation](../../docs/forwarder.md) "ndnping" scenario to ensure the forwarder works.

### File Server Preparation

If fileserver usage is desired, populate a directory on the traffic generator host with [prepare-fileserver.sh](prepare-fileserver.sh) script.
Sample command:

```bash
sudo bash ./prepare-fileserver.sh /tmp/ndndpdk-benchmark-fileserver
```

### Usage

1. Run `corepack pnpm install` to install dependencies.
2. Copy `sample.env` as `.env`, and then edit `.env` according to the comments within.
3. Start NDN-DPDK service instances.
4. Start SSH tunnels for reaching remote NDN-DPDK service instances, as necessary.
5. Run `corepack pnpm serve` to start the web application.
6. Visit `http://127.0.0.1:3333` (via SSH tunnel) in your browser.
7. To run the benchmark in command line instead of webpage, use the *CLI* button in the webapp.
