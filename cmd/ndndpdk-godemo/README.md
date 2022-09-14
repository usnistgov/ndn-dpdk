# ndndpdk-godemo

This command demonstrates [NDNgo library](../../ndn) features.

Use `-h` flag to view usage:

```bash
# see available subcommands and global options
ndndpdk-godemo -h

# see options of a subcommand
ndndpdk-godemo pingserver -h
```

## L3 Face API

[dump.go](dump.go) implements a traffic dump tool using [l3.Face API](../../ndn/l3).
This subcommand does not need a local forwarder.
This subcommand requires sudo privilege in order to use AF\_PACKET socket.

```bash
sudo ndndpdk-godemo dump --netif eth1

# --respond flag enables this tool to reply every Interest with a Data packet
sudo ndndpdk-godemo dump --netif eth1 --respond
```

## Endpoint API

[ping.go](ping.go) implements ndnping reachability test client and server using [endpoint API](../../ndn/endpoint).
This subcommand requires a local forwarder.
This subcommand does not need sudo privilege, but you may need to manually create `/run/ndn` directory beforehand.

```bash
# minimal
ndndpdk-godemo pingserver --name /pingdemo
ndndpdk-godemo pingclient --name /pingdemo

# with optional flags
ndndpdk-godemo --mtu 9000 --logging=false pingserver --name /pingdemo --payload 8000 --signed
ndndpdk-godemo --mtu 9000 pingclient --name /pingdemo --interval 100ms --lifetime 1000ms --verified
```

* `--name` flag (required) specifies the NDN name prefix.
  * Unlike [ndnping from ndn-tools](https://github.com/named-data/ndn-tools/tree/ndn-tools-22.02/tools/ping), this program does not automatically append a `ping` component.
* `--mtu` flag specifies the MTU of memif interface between this program and the local NDN-DPDK forwarder.
  * This flag must appear between 'ndndpdk-godemo' and the subcommand name.
* `--logging=false` flag disables logging to improve performance.
  * This flag must appear between 'ndndpdk-godemo' and the subcommand name.
  * With logging disabled, you can understand application activities through forwarder counters.
* `--payload` flag (pingserver only) specifies Content payload length in octets.
  * It's recommended to keep Data packet size (Name, Content, and other fields) under the MTU.
    Otherwise, NDNLPv2 fragmentation will be used.
* `--signed` flag (pingserver only) enables Data packet signing.
* `--interval` flag (pingclient only) sets interval between Interest transmissions.
* `--lifetime` flag (pingclient only) sets InterestLifetime.
* `--verified` flag (pingclient only) enables Data packet verification.

## Segmented Object API

[segmented.go](segmented.go) implements a file transfer utility using [segmented object API](../../ndn/segmented).
This subcommand requires a local forwarder.
This subcommand does not need sudo privilege, but you may need to manually create `/run/ndn` directory beforehand.

```bash
# generate test file and view digest
dd if=/dev/urandom of=/tmp/1GB.bin bs=1M count=1024
openssl sha256 /tmp/1GB.bin

# start producer
ndndpdk-godemo --mtu 6000 put --name /segmented/1GB.bin --file /tmp/1GB.bin --chunk-size 4096

# (on another console) run consumer and compute downloaded digest
ndndpdk-godemo --mtu 6000 get --name /segmented/1GB.bin | openssl sha256
```
