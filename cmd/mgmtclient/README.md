# ndn-dpdk/cmd/mgmtclient

This package contains a simple CLI management client, intended for debugging purpose.

`mgmtcmd.sh` constructs a JSON-RPC request, sends it to the running NDN-DPDK program (such as the forwarder) via the proxy, and displays the response in JSON format.
Execute `mgmtcmd.sh help` to show available subcommands.

`create-face.ts` creates a face and prints the FaceId on stdout.
This allows scripts to create a face then insert FIB entries with `mgmtcmd.sh fib insert`.

Due to JSON-RPC client library limitations, both of the above programs can only connect to management listener at TCP port 6345.
This differs from the default Unix socket listener used by NDN-DPDK management.
You may either:

* have NDN-DPDK listen on TCP by setting `MGMT=tcp4://127.0.0.1:6345` environment variable, or
* run a TCP-to-Unix proxy `socat TCP-LISTEN:6345,reuseaddr,fork,bind=127.0.0.1 UNIX-CONNECT:/var/run/ndn-dpdk-mgmt.sock`.
