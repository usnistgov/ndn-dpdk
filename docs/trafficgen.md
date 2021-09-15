# NDN-DPDK Traffic Generator Activation and Usage

After [installing NDN-DPDK](INSTALL.md) and starting the `ndndpdk-svc` service process, it can be activated as a traffic generator or some other role.
This page explains how to activate the NDN-DPDK service as a traffic generator, and how to perform some common operations.

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

The input dispatching method in the traffic generator requires every face to have a separate input thread.
Hence, if two faces are on the same Ethernet adapter, the Ethernet adapter must support [hardware-accelerate receive path (rxFlow)](../iface/ethface).
It is not recommended to use the traffic generator on socket faces, AF\_PACKET, or AF\_XDP Ethernet adapters.

## Start the Traffic Generator

After starting the `ndndpdk-svc` service process or container, there are two steps to start a traffic generator:

1. Activate the service process as traffic generator role.
   This prepares the service to accept traffic generator related commands.

   You must prepare a JSON document that contains traffic generator activation parameters, which must conform to the JSON schema `trafficgen.schema.json` (installed in `/usr/local/share/ndn-dpdk` and [available online](https://ndn-dpdk.ndn.today/schema/trafficgen.schema.json)).
   You can use the `ndndpdk-ctrl activate-trafficgen` command, or programmatically activate the traffic generator via GraphQL `activate` mutation with `trafficgen` input.

2. Start a traffic generator with traffic patterns.
   This requests the service process to create a face, allocate producer and consumer threads, and launch them with the provided traffic patterns.

   You must prepare a JSON document that contains traffic patterns configuration, which must conform to the JSON schema `gen.schema.json` (installed in `/usr/local/share/ndn-dpdk` and [available online](https://ndn-dpdk.ndn.today/schema/gen.schema.json)).
   You can use the `ndndpdk-ctrl start-trafficgen` command, or programmatically start a traffic generator via GraphQL `startTrafficGen` mutation.

### Authoring Parameters in TypeScript

NDN-DPDK provides TypeScript definitions to help with authoring the parameters.
You may install the NPM package from `/usr/local/share/ndn-dpdk/ndn-dpdk.npm.tgz` (built from [js](../js) directory), and then construct an object of `ActivateGenArgs` type for activation and `TgConfig` for starting.

[docs/activate](activate) is a sample TypeScript project that generates the parameters.
You can follow a similar procedure as [forwarder activation and usage](forwarder.md) to use this sample.
`gen-args.ts` contains activation parameters.
`gen-config.ts` contains traffic patterns configuration.

### Commonly Used Activation Parameters

**.eal.pciDevices** is a list of Ethernet adapters you want to use, written as PCI addresses.
To find the PCI addresses of available Ethernet adapters, run `dpdk-devbind.py --status-dev net`.
You should list the PCI address of every Ethernet adapter that you intend to use.

**.mempool.DIRECT.dataroom** is the size of each packet buffer.
The maximum MTU supported by the traffic generator is this dataroom minus 128 (`RTE_PKTMBUF_HEADROOM` constant).

### Commonly Used Traffic Pattern Configuration Parameters

**.face.scheme** should be one of "ether", "udpe", or "vxlan".
Other schemes will not work properly.

**.face.portConfig.mtu** is the Ethernet adapter MTU, which includes Ethernet/UDP/VXLAN headers.
It cannot exceed the limitation described in `.mempool.DIRECT.dataroom` activation parameter.

**.face.mtu** is the face MTU, which excludes Ethernet/UDP/VXLAN headers.
It cannot exceed `.face.portConfig.mtu` minus header length.

**.producer.patterns\[\].prefix** is the name prefix for matching Interests.
The producer selects the first pattern whose prefix matches an incoming Interest, and does not perform longest prefix match.

**.producer.patterns\[\].replies\[\].payloadLen** is the content payload length.
The total packet size, including NDNLPv2 header, name, payload, and signature, cannot exceed `.face.mtu`, or the packet would be dropped.

**.producer.patterns\[\].replies\[\].weight** is the probability of selecting a reply among all replies defined under a pattern.
If you define two replies with weights 9 and 1, the first reply has a 90% probability of being selected.

**.consumer.interval** is the average interval between two Interests sent from the consumer.
For efficiency, the consumer sends Interests in bursts, but the average interval would be as configured.

**.consumer.patterns\[\].weight** is the probability of selecting a pattern among all patterns.
If you define two patterns with weights 2 and 3, 40% of the outgoing Interests will be generated by the first pattern.

## Control the Traffic Generator

When you start a traffic generator, the `ndndpdk-ctrl start-trafficgen` command or `startTrafficGen` mutation returns a JSON object that contains the ID of the traffic generator.
You may use `ndndpdk-ctrl watch-trafficgen` command or GraphQL `watchTrafficGen` subscription to receive periodical updates of traffic generator counters.
You may use `ndndpdk-ctrl stop-trafficgen` command or GraphQL `delete` mutation to stop the traffic generator.

Sample commands:

```bash
TGID=$(npm run -s gen-config | ndndpdk-ctrl start-trafficgen | tee /dev/stderr | jq -r '.id')
ndndpdk-ctrl watch-trafficgen --id $TGID
ndndpdk-ctrl stop-trafficgen --id $TGID
```
