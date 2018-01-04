# ndnpktcopy-dpdk

This program reads NDN packets from one interface, and writes them on one or more interfaces.

## Usage

```
sudo ndnpktcopy-dpdk EAL-ARGS -- -in FACE [-out FACE,FACE] [-dump] [-cnt DURATION]
```

**-in** specifies the input face (required).
**-out** specifies output faces, separated by comma; duplicates are disallowe.
**-dump** prints type and name of every NDN packet.
**-cnt** specifies the interval between printing face counters.

*FACE* is one of the following:

* `dev://net_pcap0`: DPDK ethdev `net_pcap0`
* `udp://10.0.2.1:6363`: UDP socket
* `tcp://10.0.2.1:6363`: TCP socket
