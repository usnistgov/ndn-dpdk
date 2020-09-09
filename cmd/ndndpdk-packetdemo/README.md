# ndndpdk-packetdemo

This program is a demonstration of low-level APIs in [ndn-dpdk/ndn](../../ndn) and related packages.

## Features

* Traffic dumper: `-dump` flag.
* Consumer: `-transmit` flag; add `-dump` to see Data/Nack replies.
* Producer: `-respond` flag; add `-dump` to see incoming Interests.
* Communicate on a network interface using [package afpacket](../../ndn/packettransport/afpacket): specify network interface name with `-i` flag.
* Connect to a local NDN-DPDK forwarder using [package memiftransport](../../ndn/memiftransport): omit `-i` flag.

## Examples

```
# dump: print received packet names seen on eth1
sudo ndndpdk-packetdemo -i eth1 -dump

# consumer: send Interests via local forwarder
sudo ndndpdk-packetdemo -transmit 1ms -prefix /packetdemo -dump

# producer: respond to every Interest with Data
sudo ndndpdk-packetdemo -i eth1 -respond -payloadlen 1000

# producer: add route on local forwarder and respond to Interests
sudo ndndpdk-packetdemo -respond -register /packetdemo
```

Execute `ndndpdk-packetdemo -h` to see additional flags.
