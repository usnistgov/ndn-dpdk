# ndn-dpdk/cmd/mgmtclient

This package contains a simple CLI management client, intended for debugging purpose.

`mgmtproxy.sh` establishes a TCP-to-Unix proxy so that [jayson](https://www.npmjs.com/package/jayson) can connect to it.
Note: although `jayson` can connect to a Unix socket, it would send an HTTP request instead of raw JSON-RPC 2.0 request, and that would not work.
Execute `sudo mgmtproxy.sh start` to start the proxy.

`mgmtcmd.sh` constructs a JSON-RPC request, sends it to the running NDN-DPDK program (such as the forwarder) via the proxy, and displays the response in JSON format.
