# ndnpktcopy-dpdk

This program reads NDN packets from one interface, and writes them on one or more interfaces.

## Usage

```
sudo ndnpktcopy-dpdk EAL-ARGS -- -in FACE -out FACE,FACE
```

*FACE* can be specified as one of the following:

* `dev://net_pcap0`: DPDK ethdev `net_pcap0`
* `udp://10.0.2.1:6363`: UDP socket
* `tcp://10.0.2.1:6363`: TCP socket

It is an error to specify the same face more than once.
