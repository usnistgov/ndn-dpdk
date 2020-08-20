# ndn-dpdk/cmd/mgmtclient

This directory contains a simple CLI management client, intended for debugging purposes.

`mgmtcmd.sh` constructs a JSON-RPC request, sends it to the running NDN-DPDK program (such as the forwarder), and displays the response in JSON format.
Execute `mgmtcmd.sh help` to show the available subcommands.

Due to a limitation of the JSON-RPC client, this script can only connect to the management listener on TCP port 6345.
This differs from the default Unix socket listener used by NDN-DPDK management.
As a workaround, you can either:

* have NDN-DPDK listen on a TCP port by setting the `MGMT` environment variable, e.g., `export MGMT=tcp4://127.0.0.1:6345`; or
* run a TCP-to-Unix proxy, e.g., `socat TCP-LISTEN:6345,reuseaddr,fork,bind=127.0.0.1 UNIX-CONNECT:/var/run/ndn-dpdk-mgmt.sock`.
