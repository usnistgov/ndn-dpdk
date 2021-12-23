# ndndpdk-godemo

This command demonstrates [NDNgo library](../../ndn) features.

Use `-h` flag to view usage:

```bash
# see available subcommands and global options
ndndpdk-godemo -h

# see options of a subcommand
ndndpdk-godemo pingserver -h
```

All subcommands require sudo privilege in order to use AF\_PACKET socket or memif interface.

## L3 Face API

[dump.go](dump.go) implements a traffic dump tool using [l3.Face API](../../ndn/l3).
This example does not need a local forwarder.

```bash
sudo ndndpdk-godemo dump --netif eth1

# --respond flag enables this tool to reply every Interest with a Data packet
sudo ndndpdk-godemo dump --netif eth1 --respond
```

## Endpoint API

[ping.go](ping.go) implements ndnping reachability test client and server using [endpoint API](../../ndn/endpoint).
This example requires a local forwarder.

```bash
# minimal
sudo ndndpdk-godemo pingserver --name /pingdemo
sudo ndndpdk-godemo pingclient --name /pingdemo

# with optional flags
sudo ndndpdk-godemo --mtu 9000 pingserver --name /pingdemo --payload 8000 --signed
sudo ndndpdk-godemo --mtu 9000 pingclient --name /pingdemo --interval 100ms --lifetime 1000ms --verified
```

* `--name` flag (required) specifies the NDN name prefix.
  * Unlike [ndnping from ndn-tools](https://github.com/named-data/ndn-tools/tree/ndn-tools-0.7.1/tools/ping), this program does not automatically append a `ping` component.
* `--mtu` flag specifies the MTU of memif interface between this program and the local NDN-DPDK forwarder.
  * This flag must appear between 'ndndpdk-godemo' and the subcommand name.
* `--payload` flag (pingserver only) specifies Content payload length in octets.
  * It's recommended to keep Data packet size (Name, Content, and other fields) under the MTU.
    Otherwise, NDNLPv2 fragmentation will be used.
* `--signed` flag (pingserver only) enables Data packet signing.
* `--interval` flag (pingclient only) sets interval between Interest transmissions.
* `--lifetime` flag (pingclient only) sets InterestLifetime.
* `--verified` flag (pingclient only) enables Data packet verification.

## Segmented Object API

[segmented.go](segmented.go) implements a file transfer utility using [segmented object API](../../ndn/segmented).
This example requires a local forwarder.

```bash
# generate test file and view digest
dd if=/dev/urandom of=/tmp/1GB.bin bs=1M count=1024
openssl sha256 /tmp/1GB.bin

# start producer
sudo ndndpdk-godemo --mtu 6000 put --name /segmented/1GB.bin --file /tmp/1GB.bin --chunk-size 4096

# (on another console) run consumer and compute downloaded digest
sudo ndndpdk-godemo --mtu 6000 get --name /segmented/1GB.bin | openssl sha256
```
