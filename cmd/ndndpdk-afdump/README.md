# ndndpdk-afdump

This program receives packets on a network interface, optionally prints packet names, and optionally responds to every Interest with Data.
It demonstrates how to use [ndn-dpdk/ndn/afpackettransport](../../ndn/afpackettransport) package.

## Usage

```
sudo ndndpdk-afdump -i eth1 -v

sudo ndndpdk-afdump -i eth1 -respond
```
