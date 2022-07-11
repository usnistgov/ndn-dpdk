# NDN-DPDK Traffic Generator Activation and Usage

After [installing NDN-DPDK](INSTALL.md) and starting the `ndndpdk-svc` service process, it can be activated as a traffic generator or some other role.
This page explains how to activate the NDN-DPDK service as a traffic generator, and how to perform some common operations.

See [interactive benchmark](../sample/benchmark) for a web application that performs forwarder throughput benchmark using the traffic generator.

## Features and Limitations

The NDN-DPDK traffic generator is a program that transmits and receives NDN packets as fast as possible on a network interface.
It is designed to operate directly on Ethernet adapters, comparable a hardware appliance.
It does not require a local forwarder.

While IP/Ethernet traffic generators are available in both hardware and software formats, they do not understand NDN packet semantics and are unsuitable for NDN traffic generator.
For instance, an NDN producer must receive incoming Interests and respond with matching names, but a generic IP/Ethernet traffic generator cannot extract Interest names.
Hence, it is necessary to use a NDN traffic generator for testing an NDN network such as the NDN-DPDK forwarder.

You can create one or more traffic generators within an NDN-DPDK service instance activated as traffic generator role.
Each traffic generator contains a producer, a consumer, or both, attached to a network interface.
There are two choices for the producer, simple producer or file server.
There are two choices for the consumer, simple consumer or congestion aware fetcher.

If you create multiple traffic generators, each must be associated with a different face.
Each traffic generator is given dedicated packet queues and CPU lcores.
Generally, each face requires 1 input thread, 1 output thread, 2 consumer threads, and 1 producer thread.
They should be on the same NUMA socket as the Ethernet adapter.
Packet buffer mempools are shared among traffic generators on the same NUMA socket.

### Simple Producer

The simple [producer](../app/tgproducer) responds to Interests according to flexible, randomized traffic patterns.
As an example, it may be configured with these traffic patterns:

* If the Interest name starts with `/D`, reply with Data packet with 1000-octet payload.
* If the Interest name starts with `/T`, with 10% probability the packet is dropped, otherwise it is replied with a Data packet.

It maintains packet counters for each traffic pattern.

### File Server

The [file server](../app/fileserver) serves content from a local filesystem.
See [NDN-DPDK file server](fileserver.md) for more information.

### Simple Consumer

The simple [consumer](../app/tgconsumer) sends Interests at a fixed interval according to flexible, randomized traffic patterns.
As an example, it may be configured with these traffic patterns:

* Send one Interest every 100 microseconds on average.
* With 30% probability, send an Interest named `/A` followed by an increasing sequence number *seqA*, set the CanBePrefix flag.
* With 60% probability, send an Interest named `/B` followed by an increasing sequence number *seqB*.
* With 10% probability, send an Interest named `/B` followed by *seqB-9000*, allowing a potential cache hit.

It maintains packet counters for each traffic pattern, and collects round trip time statistics for Data replies.

### Congestion Aware Fetcher

The [congestion aware fetcher](../app/fetch) retrieves segmented objects such as files.
It supports a congestion control algorithms and has basic reaction to congestion control signals.
It can either write retrieved Data payload to a file for "real" file retrieval, or discard retrieved packets for emulating file retrieval traffic pattern without incurring disk I/O overhead.

## Start the Traffic Generator

After starting the `ndndpdk-svc` service process or container, follow these steps to start a traffic generator:

1. Activate the service process as traffic generator role.
   This prepares the service to accept traffic generator related commands.

   You must prepare a JSON document that contains traffic generator activation parameters, which must conform to the JSON schema `trafficgen.schema.json` (installed in `/usr/local/share/ndn-dpdk` and [available online](https://ndn-dpdk.ndn.today/schema/trafficgen.schema.json)).
   You can use the `ndndpdk-ctrl activate-trafficgen` command, or programmatically activate the traffic generator via GraphQL `activate` mutation with `trafficgen` input.

2. Create Ethernet ports for the faces needed in traffic generators.
   See [face creation](face.md) for instructions on creating the port.

   It's recommended to create the Ethernet port with PCI driver and enable RxFlow feature.
   This gives the best performance, and allows running multiple traffic generators on the same port.

   For any other setup (non-PCI, no RxFlow, SocketFace, etc), you can only run one traffic generator.
   Creating multiple traffic generators could cause unreliable results and possibly crash.

3. Start a traffic generator with traffic patterns.
   This requests the service process to create a face, allocate producer and consumer threads, and launch them with the provided traffic patterns.

   You must prepare a JSON document that contains traffic patterns configuration, which must conform to the JSON schema `gen.schema.json` (installed in `/usr/local/share/ndn-dpdk` and [available online](https://ndn-dpdk.ndn.today/schema/gen.schema.json)).
   You can use the `ndndpdk-ctrl start-trafficgen` command, or programmatically start a traffic generator via GraphQL `startTrafficGen` mutation.

4. If the traffic generator shall communicate with a forwarder, create face and FIB entry on the forwarder.
   The traffic generator would not automatically perform prefix registration on the forwarder.

At this point, simple producer or file server is ready to receive incoming packets, and simple consumer starts sending packets.
If a congestion aware fetcher is defined, it is ready to accept fetch task submissions, see "use the congestion aware fetcher" for how to submit a fetch task.

### Authoring Parameters in TypeScript

NDN-DPDK provides TypeScript definitions to help with authoring the parameters.
You may install the NPM package from `/usr/local/share/ndn-dpdk/ndn-dpdk.npm.tgz` (built from [js](../js) directory), and then construct an object of `ActivateGenArgs` type for activation and `TgConfig` for starting.

[NDN-DPDK activation sample](../sample/activate) is a sample TypeScript project that generates the parameters.

### Commonly Used Activation Parameters

**.mempool.DIRECT.dataroom** is the size of each packet buffer.
The maximum Ethernet port MTU supported by the traffic generator is this dataroom minus DPDK mbuf headroom (`RTE_PKTMBUF_HEADROOM`, normally 128) and Ethernet+VLAN header length (18).
For example, to support MTU=9000, this must be set to at least 9146.

### Commonly Used Traffic Pattern Configuration Parameters

**.face** should be the locator of an Ethernet-based face that can be created on an Ethernet port that is already created.
See [face creation](face.md) for locator syntax.

**.producer.patterns\[\].prefix** is the name prefix for matching Interests.
The producer selects the first pattern whose prefix matches an incoming Interest, and does not perform longest prefix match.

**.producer.patterns\[\].replies\[\].payloadLen** is the content payload length.
The total packet size, including NDNLPv2 header, name, payload, and signature, cannot exceed the face MTU, or the packet would be dropped.

**.producer.patterns\[\].replies\[\].weight** is the probability of selecting a reply among all replies defined under a pattern.
If you define two replies with weights 9 and 1, the first reply has a 90% probability of being selected.

**.consumer.interval** is the average interval between two Interests sent from the consumer.
For efficiency, the consumer sends Interests in bursts, but the average interval would be as configured.

**.consumer.patterns\[\].weight** is the probability of selecting a pattern among all patterns.
If you define two patterns with weights 2 and 3, 40% of the outgoing Interests will be generated by the first pattern.

**.fetcher.nTasks** is the maximum number of active fetch tasks on a congestion aware fetcher.

## Control the Traffic Generator

When you start a traffic generator, the `ndndpdk-ctrl start-trafficgen` command or GraphQL `startTrafficGen` mutation returns a JSON object that contains the ID of the traffic generator.
You may use `ndndpdk-ctrl watch-trafficgen` command or GraphQL `watchTrafficGen` subscription to receive periodical updates of traffic generator counters.
You may use `ndndpdk-ctrl stop-trafficgen` command or GraphQL `delete` mutation to stop the traffic generator.

Sample commands:

```bash
TGID=$(corepack pnpm -s start gen-config.ts | ndndpdk-ctrl start-trafficgen | tee /dev/stderr | jq -r .id)
ndndpdk-ctrl watch-trafficgen --id $TGID
ndndpdk-ctrl stop-trafficgen --id $TGID
```

### Use the Congestion Aware Fetcher

When a congestion aware fetcher is created, it does not immediately start sending Interests.
Instead, it becomes ready to accept fetch task submissions.
Each fetch task contains a name prefix to fetch from, and optionally an output filename to write the retrieved payload.
This design allows a fetcher to fetch multiple segmented objects without needing to restart the traffic generator.

You may start a fetch task with the `ndndpdk-ctrl start-fetch` command or GraphQL `fetch` mutation.
It returns a JSON object that contains the ID of the fetch task context.
If the retrieved segmented object is being written to an output file, you must set `--segment-end` and `--segment-length`; otherwise, the output file would become corrupted.

You may use `ndndpdk-ctrl watch-fetch` command or GraphQL `fetchCounters` subscription to receive periodical updates of fetcher counters.
When the `finished` field becomes non-null, the fetch task has finished, i.e., reached the last segment number either defined in the fetch task or indicated in FinalBlockId field of Data packets.

You may use `ndndpdk-ctrl stop-fetch` command or GraphQL `delete` mutation to stop a fetch task.
This step is necessary even if the fetch task has finished.

Sample commands:

```bash
FID=fad42ea2  # set to .fetcher.id when starting a traffic generator

# fetch segmented object and write to file
TASKID=$(ndndpdk-ctrl start-fetch --fetcher $FID --name /P/0 --segment-begin 0 --segment-end 1000 \
         --segment-len 4096 --filename /tmp/P0.bin | tee /dev/stderr | jq -r .id)
# watch the progress; --auto-stop may be used with a finite-sized segmented object (--segment-end specified)
# to automatically stop the task upon finish, replacing stop-fetch command
ndndpdk-ctrl watch-fetch --id $TASKID --auto-stop

# or, generate file retrieval like traffic but don't write to file
TASKID=$(ndndpdk-ctrl start-fetch --fetcher $FID --name /P/0 | tee /dev/stderr | jq -r .id)
# watch the progress; --auto-stop is ineffective because the segmented object is infinite-sized
ndndpdk-ctrl watch-fetch --id $TASKID
# abort the fetch task
ndndpdk-ctrl stop-fetch --id $TASKID
```
