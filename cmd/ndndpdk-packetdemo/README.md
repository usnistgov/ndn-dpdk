# ndndpdk-packetdemo

This program is a demonstration of [ndn-dpdk/ndn/packettransport/afpacket](../../ndn/packettransport/afpacket) package.
It can be used in several ways:

```
# dump: print received packet names
sudo ndndpdk-packetdemo -i eth1 -dump

# consumer: send Interests
#   add -dump flag to see received Data
sudo ndndpdk-packetdemo -i eth1 -transmit 1ms

# producer: respond to every Interest with Data
#   add -dump flag to see received Interests
sudo ndndpdk-packetdemo -i eth1 -respond -payloadlen 1000
```

Execute `ndndpdk-packet-demo -h` to see additional flags.
