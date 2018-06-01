# ndn-dpdk/appinit

This package implements program initialization procedures.
Most procedures are designed to terminate the process (via `log.Fatal`) if an error occurs.

## EAL (eal.go)

`InitEal` initializes DPDK's EAL.
It is required before calling any other function that depends on DPDK.

`Launch` and `MustLaunch` launch an lcore on specified NUMA socket.

**LCoreReservations** type allows reserving lcore(s) for launching later.

## Memory Pools (mempool.go)

`RegisterMempool` registers a template for mempool creation.
A number of templates have been registered automatically.

`MakePktmbufPool` creates a mempool on specified NUMA socket based on a template.

## Face Creation (face.go)

`GetFaceTable` returns the global **FaceTable** instance.

`NewFace*` functions allow creating faces from FaceUri.

## Initialization Configuration (init-config.go)

`DeclareInitConfigFlag` accepts structured configuration from either the command line or a file, to initialize mempool templates and others.
This is intended for options that must be specified during initialization and are more or less fixed for a node.
Options that are modifiable during runtime, such as FIB entries, should be exposed via management RPC server.
Options that change between program executions, such as log levels and producer name prefix, should appear as environment variables or simple command line flags.

## Management (mgmt.go)

`RegisterMgmt` registers a management module.

`StartMgmt` launches the management RPC server.
