# ndnpktcopy-dpdk

This program reads NDN packets from one interface, and writes them on one or more interfaces.

## Usage

```
sudo ndnpktcopy-dpdk EAL-ARGS -- -faces FACE,FACE [-pair|-all|-oneway] [-dump] [-cnt DURATION]
```
**-faces** specifies input and output faces, separated by comma.
**-pair** forwards packets between face at index 0-1, 2-3, 4-5, etc (default).
**-all** copies packets among all faces.
**-oneway** receives packets on first face and sends on all other faces.
**-dump** prints type and name of every NDN packet.
**-cnt** specifies the interval between printing face counters.

*FACE* is one of the following:

* `dev://net_pcap0`: DPDK ethdev `net_pcap0`
* `udp://10.0.2.1:6363`: UDP socket
* `tcp://10.0.2.1:6363`: TCP socket
