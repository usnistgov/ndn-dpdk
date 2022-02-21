# NDN-DPDK Traffic Generator Activation and Usage

After [installing NDN-DPDK](INSTALL.md) and starting the `ndndpdk-svc` service process, it can be activated as a traffic generator or some other role.
This page explains how to activate the NDN-DPDK service as a traffic generator, and how to perform some common operations.

See [interactive benchmark](../sample/benchmark) for a web application that performs forwarder throughput benchmark using the traffic generator.

## Features and Limitations

The NDN-DPDK traffic generator is a program that transmits and receives NDN packets as fast as possible on a network interface.
It is designed to operate directly on Ethernet adapters, similar as a hardware appliance; it does not require a local forwarder.

You can attach producer, consumer, or both to a network interface.
Compare to an IP/Ethernet traffic generator, the NDN-DPDK traffic generator understands NDN packet semantics.
For example, an NDN producer must receive incoming Interests and respond with matching names, which is not supported by IP/Ethernet traffic generators.

The traffic generator supports flexible, randomized traffic patterns.
For example, a producer may be configured with these traffic patterns:

* If the Interest name starts with `/D`, reply with Data packet with 1000-octet payload.
* If the Interest name starts with `/T`, with 10% probability the packet is dropped, otherwise it is replied with a Data packet.

Likewise, a consumer may be configured with these traffic patterns:

* Send one Interest every 100 microseconds on average.
* With 30% probability, send an Interest named `/A` followed by an increasing sequence number *seqA*, set the CanBePrefix flag.
* With 60% probability, send an Interest named `/B` followed by an increasing sequence number *seqB*.
* With 10% probability, send an Interest named `/B` followed by *seqB-9000*, allowing a potential cache hit.

The traffic generator maintains packet counters of each traffic pattern.
The consumer also collects round trip time statistics for Data replies.

You can start multiple traffic generators on different faces within the same `ndndpdk-svc` service process, subject to available hardware resources.
Traffic generator associated with each face is given dedicated packet queues and CPU lcores.
Generally, each face requires 1 input thread, 1 output thread, 2 consumer threads, and 1 producer thread.
They should be on the same NUMA socket as the Ethernet adapter.

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

## Control the Traffic Generator

When you start a traffic generator, the `ndndpdk-ctrl start-trafficgen` command or GraphQL `startTrafficGen` mutation returns a JSON object that contains the ID of the traffic generator.
You may use `ndndpdk-ctrl watch-trafficgen` command or GraphQL `watchTrafficGen` subscription to receive periodical updates of traffic generator counters.
You may use `ndndpdk-ctrl stop-trafficgen` command or GraphQL `delete` mutation to stop the traffic generator.

Sample commands:

```bash
TGID=$(corepack pnpm start -s gen-config.ts | ndndpdk-ctrl start-trafficgen | tee /dev/stderr | jq -r '.id')
ndndpdk-ctrl watch-trafficgen --id $TGID
ndndpdk-ctrl stop-trafficgen --id $TGID
```
