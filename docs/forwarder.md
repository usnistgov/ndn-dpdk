# NDN-DPDK Forwarder Activation and Usage

After [installing NDN-DPDK](INSTALL.md) and starting the `ndndpdk-svc` service process, it can be activated as a forwarder or some other role.
This page explains how to activate the NDN-DPDK service as a forwarder, and how to perform some common operations.

## Activate the Forwarder

Before attempting to activate the forwarder, make sure you have configured hugepages and PCI driver bindings, as described on [installation guide](INSTALL.md) "usage" section.

The `ndndpdk-ctrl activate-forwarder` command sends a command to the `ndndpdk-svc` service process to activate it as a forwarder.
This command reads, from standard input, a JSON document that contains forwarder activation parameters.
The JSON document must conform to the JSON schema `forwarder.schema.json` (installed in `/usr/local/share/ndn-dpdk` and [available online](https://ndn-dpdk.ndn.today/schema/forwarder.schema.json)).

If you have prepared the required JSON document (e.g. using the TypeScript definitions, as described below), you can activate the forwarder with:

```bash
ndndpdk-ctrl activate-forwarder < fw-args.json
```

You can also programmatically activate the forwarder via GraphQL using the `activate` mutation, passing the JSON document as the `forwarder` input.

The `ndndpdk-svc` service process can be activated only once.
You must restart the systemd service or Docker container to activate again as a different role or with different parameters.

### What's My Activation Parameters?

All fields in the forwarder activation parameters are optional.
You can pass an empty JSON object `{}` to activate the forwarder with default settings, which would work if your system has sufficient CPU and memory described in [hardware known to work](hardware.md) "CPU and memory" section.
If this is your first time setting up NDN-DPDK, it is strongly recommended to use a system with the required resources.

The optimal activation parameters to take full advantage of your system is hardware dependent.
See [hardware known to work](hardware.md) and [performance tuning](tuning.md) for hints.

### Authoring Activation Parameters in TypeScript

NDN-DPDK provides TypeScript definitions to help with authoring the activation parameters.
Commonly used options have description or links to the corresponding Go documentation.
You may install the NPM package from `/usr/local/share/ndn-dpdk/ndn-dpdk.npm.tgz` (built from [js](../js) directory), and then construct an object of `ActivateFwArgs` type.

[NDN-DPDK activation sample](../sample/activate) is a sample TypeScript project that generates the activation parameters.

### Commonly Used Activation Parameters

This section explains some commonly used parameters.

**.eal.cores** is a list of CPU cores allocated to DPDK.
NDN-DPDK also honors CPU affinity configured in systemd or Docker, see [performance tuning](tuning.md) "CPU isolation".

**.mempool.DIRECT.dataroom** is the size of each packet buffer.
The maximum Ethernet port MTU supported by the forwarder is this dataroom minus DPDK mbuf headroom (`RTE_PKTMBUF_HEADROOM`, normally 128) and Ethernet+VLAN header length (18).
For example, to support MTU=9000, this must be set to at least 9146.

**.mempool.DIRECT.capacity** is the maximum quantity of packet buffers on each NUMA socket.
Every packet received by the forwarder and not yet released, including those buffered in the PIT or cached in the CS, occupies one of the packet buffers.
Therefore, the capacity must be large enough to accommodate all the queues, PIT entries, and in-memory Data packets cached in the CS; otherwise, if the capacity is too small, Ethernet adapters will eventually stop receiving packets due to lack of packet buffers.
This setting also has great impact on the RAM usage of the forwarder: if it's too large, the forwarder may fail to activate due to insufficient hugepage memory.

**.mempool.INDIRECT.capacity** is the maximum quantity of indirect entries on each NUMA socket.
Indirect entries are used to reference (part of) an existing packet buffer, which are used in various data structures and during packet transmission.
It's recommended to set this to the same as `.mempool.DIRECT.capacity`.

**.fib.capacity** is the maximum quantity of FIB entries.

**.fib.startDepth** is the *M* parameter in [2-stage LPM](https://doi.org/10.1109/ANCS.2013.6665203) algorithm.
It should be set to the 90th percentile of the anticipated number of name components in FIB entry names.

**.pcct.pcctCapacity** is the maximum quantity of PCCT entries in each forwarding thread.
This limits the combined quantity of PIT entries and CS entries in a forwarding thread.

**.pcct.csMemoryCapacity** is the maximum quantity of direct in-memory CS entries in each forwarding thread.
Each direct in-memory CS entry contains a Data packet and occupies a packet buffer, and can be found with Interests having the same name as the Data.
This capacity, multiplied by the number of forwarding threads, is roughly equivalent to the "CS capacity" in other forwarders.

**.pcct.csIndirectCapacity** is the maximum quantity of indirect CS entries in each forwarding thread.
Indirect CS entries enable prefix match lookups in the CS.
Each indirect CS entry is a pointer to a direct CS entry, but does not contain a Data packet by itself and thus does not occupy a packet buffer.
In most cases, it's recommended to set this to the same as `.pcct.csMemoryCapacity`.
If the majority of traffic in your network is exact match only, you may set a smaller value.

## Sample Scenario: ndnping

This section guides through face creation and FIB entry insertion commands, in order to complete a simple `ndnping`.
To try this scenario, you need:

* two hosts each equipped with an Ethernet adapter
* a direct attach cable connecting the two Ethernet adapters
* NDN-DPDK forwarders activated on both hosts

The hosts are labeled *A* and *B*.
When you read the example commands, make sure to use them on the correct host.

The `ndndpdk-ctrl` command line tool is used throughout the scenario.
You can run any `ndndpdk-ctrl` subcommand with `-h` flag to see available command line flags, such as `ndndpdk-ctrl create-ether-face -h`.

### Create Ethernet Faces

[Face creation](face.md) page describes that there are two steps in creating an Ethernet-based face:

1. Create an Ethernet port on the desired Ethernet adapter.
2. Create an Ethernet-based face on the Ethernet port.

The `ndndpdk-ctrl create-eth-port` command creates an Ethernet port.
It returns a JSON object that contains the DPDK device name and local MAC address of the Ethernet adapter.

The `ndndpdk-ctrl create-ether-face` command creates an Ethernet face.
It returns a JSON object that contains an `id` property, whose value is an opaque identifier of the face.

NDN-DPDK forwarder does not automatically create new faces when it receives incoming traffic from an unknown source.
Therefore, when you interconnect two NDN-DPDK forwarders, it's necessary to create faces on both forwarders.

Example command and output:

```shell
A $ ndndpdk-ctrl create-eth-port --pci 04:00.0 --mtu 9000
{"id":"1276dc31","macAddr":"02:00:00:00:00:01","name":"0000:04:00.0","numaSocket":1,"rxImpl":"RxTable"}
# to use '--mtu 9000', .mempool.DIRECT.dataroom in activation parameters should be at least 9146

B $ ndndpdk-ctrl create-eth-port --pci 06:00.0 --mtu 9000
{"id":"a6a35a10","macAddr":"02:00:00:00:00:02","name":"0000:06:00.0","numaSocket":1,"rxImpl":"RxTable"}

A $ ndndpdk-ctrl create-ether-face --local 02:00:00:00:00:01 --remote 02:00:00:00:00:02
{"id":"286d21ff"}

B $ ndndpdk-ctrl create-ether-face --local 02:00:00:00:00:02 --remote 02:00:00:00:00:01
{"id":"31bfaaa9"}
```

You can programmatically create Ethernet port and face via GraphQL using `createEthPort` and `createFace` mutations.
As a starting point, you may see the GraphQL operation used by a `ndndpdk-ctrl` command by adding `--cmdout` flag, such as:

```bash
ndndpdk-ctrl --cmdout create-eth-port --pci 04:00.0 --mtu 9000
ndndpdk-ctrl --cmdout create-ether-face --local 02:00:00:00:00:01 --remote 02:00:00:00:00:02
# --cmdout flag must appear between 'ndndpdk-ctrl' and the subcommand name,
# it is supported on all subcommands
```

### Insert a FIB Entry

The `ndndpdk-ctrl insert-fib` command inserts or overwrites a FIB entry.
It returns a JSON object that contains an `id` property, whose value is an opaque identifier of the FIB entry.

Example command and output:

```shell
A $ ndndpdk-ctrl insert-fib --name /example/P --nexthop 286d21ff
{"id":"5aa50b21"}
```

You can programmatically insert a FIB entry via GraphQL using the `insertFibEntry` mutation.

### Start the Application

Part of the NDN-DPDK repository is [NDNgo](../ndn), a minimal NDN application development library compatible with NDN-DPDK.
Its demo program, [command ndndpdk-godemo](../cmd/ndndpdk-godemo), contains a simple ndnping application.

You can start the producer and the consumer as follows:

```shell
B $ ndndpdk-godemo pingserver --name /example/P
2022/05/05 14:54:17 uplink opened, state is down
2022/05/05 14:54:17 uplink state changes to up
2022/05/05 14:54:18 /8=example/8=P/8=0E0344249FD27C3A[F]
2022/05/05 14:54:18 /8=example/8=P/8=0E0344249FD27C3B[F]
2022/05/05 14:54:19 /8=example/8=P/8=0E0344249FD27C3C[F]
2022/05/05 14:54:19 /8=example/8=P/8=0E0344249FD27C3D[F]
2022/05/05 14:54:19 /8=example/8=P/8=0E0344249FD27C3E[F]
2022/05/05 14:54:19 /8=example/8=P/8=0E0344249FD27C3F[F]
2022/05/05 14:54:40 uplink state changes to down
2022/05/05 14:54:40 uplink closed, error is <nil>

A $ ndndpdk-godemo pingclient --name /example/P
2022/05/05 14:54:18 uplink opened, state is down
2022/05/05 14:54:18 uplink state changes to up
2022/05/05 14:54:18 100.00% D 0E0344249FD27C3A   1294us
2022/05/05 14:54:18 100.00% D 0E0344249FD27C3B   1685us
2022/05/05 14:54:19 100.00% D 0E0344249FD27C3C    710us
2022/05/05 14:54:19 100.00% D 0E0344249FD27C3D    643us
2022/05/05 14:54:19 100.00% D 0E0344249FD27C3E   1182us
2022/05/05 14:54:19 100.00% D 0E0344249FD27C3F   1975us
2022/05/05 14:54:19 uplink state changes to down
2022/05/05 14:54:19 uplink closed, error is <nil>
```

The consumer prints, among other fields, the percentage of satisfied Interests and the last round-trip time.
See [command ndndpdk-godemo](../cmd/ndndpdk-godemo) for additional options such as MTU and Data payload length.

### List Faces and View Face Counters

The `ndndpdk-ctrl list-face` command returns a list of faces.

The `ndndpdk-ctrl get-face` command retrieves information about a face, including several face counters.
Observing face counter changes while the application is running is an effective way to identify where packet loss is occurring.

Example command and output:

```shell
A $ ndndpdk-ctrl get-face --id 286d21ff --cnt | jq .counters
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
