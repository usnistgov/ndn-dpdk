# NDN-DPDK File Server Activation and Usage

After [installing NDN-DPDK](INSTALL.md) and starting the `ndndpdk-svc` service process, it can be activated as a file server or some other role.
This page explains how to activate the NDN-DPDK service as a file server, and how to perform some common operations.

## Start the File Server

It is recommended to deploy the file server alongside a local NDN-DPDK forwarder.
This allows the file server to take advantage of the forwarder's content caching capability, as the file server itself does not have caching.

There are four steps to start a file server and connect it to the forwarder:

1. Start one instance of NDN-DPDK service and activate it as a forwarder.
   See [forwarder activation and usage](forwarder.md) for instructions.

2. On the forwarder, create a memif face of "server" role for the file server, and insert FIB entries to forward Interests under the file server's prefix(es).

3. Start a second instance of NDN-DPDK service.
   See [installation guide](INSTALL.md) "running multiple instances" section for requirements.

4. Activate the second instance as file server role.
   The file server will connect to the memif face previously created on the forwarder, and be ready to process Interests.

   You must prepare a JSON document that contains traffic generator activation parameters, which must conform to the JSON schema `fileserver.schema.json` (installed in `/usr/local/share/ndn-dpdk` and [available online](https://ndn-dpdk.ndn.today/schema/fileserver.schema.json)).
   You can use the `ndndpdk-ctrl activate-fileserver` command, or programmatically activate the traffic generator via GraphQL `activate` mutation with `fileserver` input.

### Authoring Parameters in TypeScript

NDN-DPDK provides TypeScript definitions to help with authoring the parameters.
You may install the NPM package from `/usr/local/share/ndn-dpdk/ndn-dpdk.npm.tgz` (built from [js](../js) directory), and then construct an object of `ActivateFileServerArgs` type.

[docs/activate](activate) is a sample TypeScript project that generates the parameters.
You can follow a similar procedure as [forwarder activation and usage](forwarder.md) to use this sample.
`fileserver-args.ts` contains activation parameters.

### Commonly Used Activation Parameters

**.face** specifies a locator for face creation within the file server.
To connect to the local NDN-DPDK forwarder, it should use "memif" scheme with "client" role.
**.face.dataroom** specifies the MTU between file server and forwarder, which is independent from the MTU of physical network links.

**.fileServer.mounts\[\].prefix** is the NDN name prefix of a mountpoint.
**.fileServer.mounts\[\].path** is the filesystem path of a mountpoint.

**.fileServer.segmentLen** is the payload length of each file segment packet (except the last segment).
It should be small enough so that the Data packet size (containing name, payload, and other fields) stays below the MTU of most network links.
Once the file server is deployed in a network, you should not change this setting without also changing `.fileServer.mounts[].prefix`, because network caches may respond with previously generated Data packets and lead to corrupted file retrieval.

**.fileServer.uringCapacity** is the io\_uring submission queue capacity.
Lower values are suitable for faster disks such as local NVMe.
If the file server will be accessing slower disks such as HDD or iSCSI, higher values (up to 32768) are recommended.

**.mempool.DIRECT** configures the mempool for incoming packets, which are expected to be Interests.
Its dataroom should accommodate the face MTU plus 128-octet headroom.
Its capacity should accommodate incoming packet queues and io\_uring capacity of all file server threads.

**.mempool.PAYLOAD** configures the mempool for file descriptors and outgoing Data packets.
Its dataroom should accommodate maximum Data packet size (see `.fileServer.segmentLen`) plus 128-octet headroom.
Its capacity should accommodate outgoing packet queues and io\_uring capacity of all file server threads.

**.eal.coresPerNuma** and **.eal.memPerNuma** allocate CPU cores and hugepage memory.
Since the file server only operates on one face, it's sufficient to allocate resources on only one NUMA socket.
The file server requires at least 4 CPU cores: 1 main lcore, 1 input thread, 1 output thread, and at least 1 file server thread.
See [performance tuning](tuning.md) "LCore Allocation" section on how to run with fewer physical CPU cores.

### Alternate Setup Methods

Instead of connecting to the local NDN-DPDK forwarder, it is possible to run a standalone file server that listens on a physical Ethernet adapter.
You may do so by setting `.face` locator to an Ethernet adapter, and including the PCI address of the Ethernet adapter in `.eal.pciDevices`.

The file server is internally implemented as a traffic generator component.
Therefore, it is possible to start a file server as part of the traffic generator.
You may do so by invoking GraphQL `startTrafficGen` mutation with a JSON document that contains `.fileServer` field.

## Sample Scenario: transfer NDN-DPDK itself

This section guides through file server setup, in order to transfer the `/usr/local/bin/ndndpdk-svc` file.
To try this scenario, you need:

* single host
* NDN-DPDK forwarder running on `http://127.0.0.1:3030` and activated
* forwarder must be able to accommodate MTU=9000, i.e. its `.mempool.DIRECT.dataroom` should be at least 9128

### Create Face and Insert FIB Entry on Forwarder

A memif face can have either "server" or "client" role.
In order to establish a connection, one peer must assume the server role, and the other peer must assume the client role.
Moreover, the server should be created before the client.
It's recommended to let the forwarder be the server, while the file server is the client.

You can create a memif face by invoking GraphQL `createFace` mutation, and then insert a FIB entry via `ndndpdk-ctrl insert-fib` command.

Example command and output:

```shell
$ FACEID=$(gq http://127.0.0.1:3030 \
    -q 'mutation createFace($locator:JSON!){createFace(locator:$locator){id}}' \
    --variablesJSON '{
      "locator": {
        "scheme": "memif",
        "socketName": "/run/ndn/fileserver.sock",
        "id": 0,
        "role": "server",
        "dataroom": 9000
      }
    }' | jq -c .data.createFace | tee /dev/stderr | jq -r .id)
Executing query... done
{"id":"7CLE6J8CP1Q8103O0Q6CPIQVR8"}

$ ndndpdk-ctrl --gqlserver http://127.0.0.1:3030 insert-fib \
    --name /fileserver --nexthop $FACEID
{"id":"6GIU001CH13976H72E0CF7OFGJOIBQ1PJM9ES"}
```

### Activate File Server in Another NDN-DPDK Service Instance

The following command starts another NDN-DPDK service instance in systemd:

```bash
sudo ndndpdk-ctrl --gqlserver http://127.0.0.1:3031 systemd start
```

If you are running [NDN-DPDK in Docker container](Docker.md), start another container from the same NDN-DPDK image.
The `/run/ndn` directory should be mounted into both containers in order to establish memif connection.
In this case, you should change `--gqlserver` flag to target the container.

The sample activation parameters given in [docs/activate](activate) may be used in this scenario.

1. Make a copy of this directory to somewhere outside the NDN-DPDK repository.
2. Run `npm install` to install dependencies.
3. Run `npm run -s fileserver-args | jq .` to see the JSON document.
   Notice in the output that `.face.role` is set to "client", opposite from the locator sent to the forwarder.
4. Run `npm run -s fileserver-args | ndndpdk-ctrl --gqlserver http://127.0.0.1:3031 activate-fileserver` to send a file server activation command.
   Notice the `--gqlserver` flag, targeting the second NDN-DPDK service instance.

### Retrieve the File

One of the mountpoints defined in the activation parameters is:

```json
{
  "prefix": "/fileserver/usr-local-bin",
  "path": "/usr/local/bin"
}
```

This means that the file `/usr/local/bin/ndndpdk-svc` will have the NDN name prefix `/fileserver/usr-local-bin/ndndpdk-svc`.

You can use the [ndncat command](https://ndnts-docs.ndn.today/typedoc/modules/cat.html) to retrieve the file, and then compare it against the original.

Example command and output:

```shell
$ export NDNTS_UPLINK=ndndpdk-udp:
$ export NDNTS_NDNDPDK_GQLSERVER=http://127.0.0.1:3030
$ npx -y -p https://ndnts-nightly.ndn.today/cat.tgz ndncat get-segmented \
    --ver=rdr /fileserver/usr-local-bin/ndndpdk-svc > /tmp/ndndpdk-svc.retrieved

$ sha256sum /usr/local/bin/ndndpdk-svc /tmp/ndndpdk-svc.retrieved
d7d68600dd33a2e344bb4e4895e10302d4f9781930b601241d5ec5aaacab6392  /usr/local/bin/ndndpdk-svc
d7d68600dd33a2e344bb4e4895e10302d4f9781930b601241d5ec5aaacab6392  /tmp/ndndpdk-svc.retrieved
```
