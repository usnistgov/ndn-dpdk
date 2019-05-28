# ndn-dpdk/mgmt

This package implements a management RPC server.
Calling process should `Register` management modules, then `Start` the server.

The RPC server uses [JSON-RPC 2.0](https://www.jsonrpc.org/specification) codec.
By default, the server listens on Unix stream socket `/var/run/ndn-dpdk-mgmt.sock`.
Sysadmin may change this path or switch to TCP through environment variable.
For example:

    MGMT=unix:///tmp/ndn-dpdk-mgmt.sock
    MGMT=tcp4://127.0.0.1:6345
    MGMT=tcp6://[::1]:6345

To disable management, set environment variable `MGMT=0`.
`Start` would have no effect after that.

The RPC server does not perform authentication.
The default Unix stream socket is reachable by root only, as a form of protection.
Client processes should start as root and open the socket, then drop root privileges if desired.

This directory also offers `make-spec.ts`, a program to create a [jrgen](https://www.npmjs.com/package/jrgen) specification file for the management API.
`make mgmtspec` writes the spec to `docs/mgmtspec.json`.
