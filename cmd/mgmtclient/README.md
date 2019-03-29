# ndn-dpdk/cmd/mgmtclient

This package contains a simple CLI management client, intended for debugging purpose.

`mgmtproxy.sh` establishes a TCP-to-Unix proxy so that [jayson](https://www.npmjs.com/package/jayson) can connect to it.
Note: although `jayson` can connect to a Unix socket, it would send HTTP requests, while [mgmt](../../mgmt/) listener only accepts raw JSON-RPC 2.0 requests, so it would not work.
Execute `sudo mgmtproxy.sh start` to start the proxy.

`mgmtcmd.sh` constructs a JSON-RPC request, sends it to the running NDN-DPDK program (such as the forwarder) via the proxy, and displays the response in JSON format.
Execute `mgmtcmd.sh help` to show available subcommands.

`create-face.ts` creates a face and prints the FaceId on stdout.
This allows scripts to create a face then insert FIB entries with `mgmtcmd.sh fib insert`.
