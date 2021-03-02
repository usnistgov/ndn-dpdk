# NDN-DPDK Forwarder Activation and Usage

After [installing NDN-DPDK](INSTALL.md) and starting the `ndndpdk-svc` service process, it can be activated as either a forwarder or a traffic generator.
This page explains how to activate the NDN-DPDK service as a forwarder, and how to perform some common operations.

## Activate the Forwarder

The `ndndpdk-ctrl activate-forwarder` command sends a command to the `ndndpdk-svc` service process to activate it as a forwarder.
This command reads, from standard input, a JSON document that contains forwarder activation parameters.
The JSON document must conform to the JSON schema `/usr/local/share/ndn-dpdk/forwarder.schema.json`.

If you have prepared the required JSON document, you can activate the forwarder with:

```bash
ndndpdk-ctrl activate-forwarder < fw-args.json
```

You can also programmatically activate the forwarder via GraphQL using the `activate` mutation, passing the JSON document as the `forwarder` input.

The `ndndpdk-svc` service process can be activated only once.
You must restart the process to activate again as a different role or with different parameters.

### Authoring Activate Parameters in TypeScript

NDN-DPDK provides TypeScript definitions to help with authoring the activation parameters.
There are description of commonly used options, or links to the godoc of the corresponding Go types.
You may install the NPM package from `/usr/local/share/ndn-dpdk/ndn-dpdk.npm.tgz`, and then construct an object of `ActivateFwArgs` type.

[docs/activate](activate) is a sample TypeScript project that generates the activation parameters.
To use the sample:

1. Make a copy of this directory to somewhere outside the NDN-DPDK repository.
2. Run `npm install` to install dependencies.
3. Open the directory in Visual Studio Code or another editor that recognizes TypeScript definitions.
   If the NDN-DPDK installation is on a remote machine, you may use the Remote-SSH plugin.
4. Open `fw-args.ts` in the editor, and make changes.
   The editor can provide hints hints on available options.
5. Run `npm run -s typecheck` to verify your arguments conforms to the TypeScript definitions.
6. Run `npm run -s fw-args | jq .` to see the JSON document.
   [jq](https://stedolan.github.io/jq/) pretty-prints the JSON document, which is optional.
7. Run `npm run -s fw-args | ndndpdk-ctrl activate-forwarder` to send a forwarder activation command.

### Commonly Used Activation Parameters

All fields in the forwarder activation parameters are optional.
You can pass an empty JSON object `{}` to activate the forwarder with default settings.
This section explains some commonly used parameters.

**.eal.cores** is a list of CPU cores allocated to DPDK.
To find the available CPU cores, run `lscpu` and look at "NUMA node? CPU(s)" line.

**.eal.pciDevices** is a list of Ethernet adapters you want to use in the forwarder, written as PCI addresses.
To find the PCI addresses of available Ethernet adapters, run `dpdk-devbind.py --status-dev net`.

**.eal.virtualDevices** is a list of DPDK virtual devices.
This can be used to enable an Ethernet adapter that is not natively supported by DPDK using the AF\_PACKET socket.
For example, `net_af_packet1,iface=eth1` enables the `eth1` Ethernet adapter as a virtual device named `net_af_packet1`.

**.mempool.DIRECT.dataroom** is the size of each packet buffer.
The maximum MTU supported by the forwarder is this dataroom minus 128 (`RTE_PKTMBUF_HEADROOM` constant).

**.mempool.DIRECT.capacity** is the maximum quantity of packet buffers on each NUMA socket.
Every packet received by the forwarder and not yet released, including those buffered in the PIT or cached in the CS, occupies one of the packet buffers.
Therefore, the capacity must be large enough to accommodate all the queues, PIT entries, and Data packets cached in the CS; otherwise, if the capacity is too small, Ethernet adapters will eventually stop receiving packets due to lack of packet buffers.
This setting also has great impact on the RAM usage of the forwarder: if it's too large, the forwarder may fail to activate due to insufficient hugepage memory.

**.mempool.INDIRECT.capacity** is the maximum quantity of indirect entries on each NUMA socket.
Indirect entries are used to reference (part of) an existing packet buffer, which are used in various data structures and during packet transmission.
It's recommended to set this to the same as `.mempool.DIRECT.capacity`.

**.fib.capacity** is the maximum quantity of FIB entries.

**.fib.startDepth** is the *M* parameter in [2-stage LPM](https://doi.org/10.1109/ANCS.2013.6665203) algorithm.
It should be set to the 90th percentile of the anticipated number of name components in FIB entry names.

**.pcct.pcctCapacity** is the maximum quantity of PCCT entries in each forwarding thread.
This limits the combined quantity of PIT entries and CS entries in a forwarding thread.

**.pcct.csDirectCapacity** is the maximum quantity of direct CS entries in each forwarding thread.
Each direct CS entry contains a Data packet and occupies a packet buffer, and can be found with Interests having the same name as the Data.
This capacity, multiplied by the number of forwarding threads, is roughly equivalent to the "CS capacity" in other forwarders.

**.pcct.csIndirectCapacity** is the maximum quantity of indirect CS entries in each forwarding thread.
Indirect CS entries enable prefix match lookups in the CS.
Each indirect CS entry is a pointer to a direct CS entry, but does not contain a Data packet by itself and thus does not occupy a packet buffer.
In most cases, it's recommended to set this to the same as `.pcct.csDirectCapacity`.
If the majority of traffic in your network is exact match only, you may set a smaller value.

## Sample Scenario: ndnping

This section guides through face creation and FIB entry insertion commands, in order to complete a simple `ndnping`.
To try this scenario, you need:

* two hosts each equipped with an Ethernet adapter.
* a direct attach cable connecting the two Ethernet adapters.
* NDN-DPDK forwarders activated on both hosts.

The hosts are labeled *A* and *B*.
When you read the example commands, make sure to use them on the correct host.

### Create Faces

The `ndndpdk-ctrl create-ether-face` command sends a command to create an Ethernet face.
You can run this command with `-h` option to see available command line arguments.
It returns a JSON object that contains an `id` property, whose value is an opaque identifier of the face.

NDN-DPDK forwarder does not automatically create new faces when it receives incoming traffic from an unknown source.
Therefore, when you interconnect two NDN-DPDK forwarders, it's necessary to create faces on both forwarders.

Example command and output:

```shell
A $ ndndpdk-ctrl create-ether-face --local 02:00:00:00:00:01 --remote 02:00:00:00:00:02
{"id":"gFmoaws197"}

B $ ndndpdk-ctrl create-ether-face --local 02:00:00:00:00:02 --remote 02:00:00:00:00:01
{"id":"e6vdYnE6G"}
```

You can programmatically create a face via GraphQL using the `createFace` mutation.
Its input is a JSON document that conforms to the JSON schema `/usr/local/share/ndn-dpdk/locator.schema.json`.
It supports more transport types and options than what's available through `ndndpdk-ctrl` commands.

### Insert a FIB Entry

The `ndndpdk-ctrl insert-fib` command inserts or overwrites a FIB entry.
You can run this command with `-h` option to see available command line arguments.
It returns a JSON object that contains an `id` property, whose value is an opaque identifier of the FIB entry.

Example command and output:

```shell
A $ ndndpdk-ctrl insert-fib --name /example/P --nexthop gFmoaws197
{"id":"JaimdtVXKn"}
```

You can programmatically insert a FIB entry via GraphQL using the `insertFibEntry` mutation.

### Start the Application

Part of the NDN-DPDK repository is [NDNgo](../ndn) that is a minimal NDN application development library compatible with NDN-DPDK.
Its demo program, [command ndndpdk-godemo](../cmd/ndndpdk-godemo), contains a simple ndnping application.

You can start the producer and the consumer as follows:

```shell
B $ sudo ndndpdk-godemo pingserver --name /example/P

A $ sudo ndndpdk-godemo pingclient --name /example/P
```

The consumer prints, among other fields, the percentage of satisfied Interests, and the round-trip time of the last Interest-Data exchange.

### List Faces and View Face Counters

The `ndndpdk-ctrl list-face` command returns a list of faces.

The `ndndpdk-ctrl get-face` command retrieves information about a face, including several face counters.
Observing face counter changes while the application is running is an effective way to identify where packet loss is occurring.

Example command and output:

```shell
A $ ndndpdk-ctrl get-face --id gFmoaws197 --cnt | jq .counters
{
  "rxData": "1024",
  "rxInterests": "0",
  "rxNacks": "0",
  "txData": "0",
  "txInterests": "1025",
  "txNacks": "0"
}
```

You can programmatically retrieve face information via GraphQL using the `faces` query.
It includes many more counters than what's available through the `ndndpdk-ctrl get-face` command.
