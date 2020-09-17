# ndndpdk-godemo

This command demonstrates [NDNgo library](../../ndn) features.

Execute `ndndpdk-godemo -h` to see available subcommands.

## L3 Face API

[dump.go](dump.go) implements a traffic dump tool using [l3.Face API](../../ndn/l3).
This example does not need a running local NDN-DPDK forwarder.

```bash
sudo ndndpdk-godemo dump --netif eth1

# --respond flag enables this tool to reply every Interest with a Data packet
sudo ndndpdk-godemo dump --netif eth1 --respond
```

## Endpoint API

[ping.go](ping.go) implements ndnping reachability test client and server using [endpoint API](../../ndn/endpoint).
This example requires a running local NDN-DPDK forwarder.

```bash
sudo ndndpdk-godemo pingserver --name /pingdemo --payload 100

sudo ndndpdk-godemo pingclient --name /pingdemo --interval 100ms --lifetime 1000ms
```
