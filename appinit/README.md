# ndn-dpdk/appinit

This package implements program initialization procedures.
Most procedures are designed to terminate the process (via `Exitf` function) if an error occurs.

## EAL (eal.go)

`InitEal` initializes DPDK's EAL.
It is required before calling any other function that depends on DPDK.

`Launch` and `LaunchRequired` launch an lcore on specified NUMA socket.

**LCoreReservations** type allows reserving lcore(s) for launching later.

## Memory Pools (mempool.go)

`RegisterMempool` registers a template for mempool creation.
A number of templates have been registered automatically.

`MakePktmbufPool` creates a mempool on specified NUMA socket based on a template.

## Face Creation (face.go)

`GetFaceTable` returns the global **FaceTable** instance.

`NewFace*` functions allow creating faces from FaceUri.

## Management RPC Server (mgmt.go)

`EnableMgmt` followed by `StartMgmt` initializes the management RPC server.
Calling process may register additional management modules on `MgmtRpcServer` variable.
This server uses JSON-RPC 2.0 codec.

By default, the RPC server listens on Unix stream socket `/var/run/ndn-dpdk-mgmt.sock`.
Sysadmin may change this path or switch to TCP through environment variable.
For example:

    MGMT=unix:///tmp/ndn-dpdk-mgmt.sock
    MGMT=tcp4://127.0.0.1:6345
    MGMT=tcp6://[::1]:6345

The Unix stream socket is reachable by root only, because there is no authentication on the RPC server.
Client processes should start as root and open the socket, then drop root privileges if desired.
