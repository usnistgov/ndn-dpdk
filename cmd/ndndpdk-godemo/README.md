# ndndpdk-godemo

This command demonstrates [NDNgo library](../../ndn) features.

Execute `ndndpdk-packetdemo -h` to see available subcommands.

## Endpoint API

[ping.go](ping.go) implements ndnping reachability test client and server using [endpoint API](../../ndn/endpoint).
This example requires a running local NDN-DPDK forwarder.

```
sudo ndndpdk-godemo pingserver --name /pingdemo --payload 100

sudo ndndpdk-godemo pingclient --name /pingdemo --interval 100ms --lifetime 1000ms
```
