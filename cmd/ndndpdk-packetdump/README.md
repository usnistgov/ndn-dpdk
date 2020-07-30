# ndndpdk-packetdump

This program receives packets on a network interface, optionally prints packet names, and optionally responds to every Interest with Data.
It demonstrates how to use [ndn-dpdk/ndn/packettransport/afpacket](../../ndn/packettransport/afpacket) package.

## Usage

```
sudo ndndpdk-packetdump -i eth1 -v

sudo ndndpdk-packetdump -i eth1 -respond
```
