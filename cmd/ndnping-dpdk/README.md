# ndnping-dpdk

This program acts as [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) client or server on specified interfaces.

## Usage

```
sudo ndnping-dpdk EAL-ARGS -- \
  [-latency] [-rtt] [-add-delay DURATION] [-nack=false] [-cnt DURATION] \
  +c FACE PREFIX PCT PREFIX PCT \
  +s FACE PREFIX PREFIX
```

**+c** defines a client on *FACE*.
*PREFIX* is a name prefix for Interests; it is recommended to use `ping` as the last name component.
*PCT* is the percentage of traffic under this prefix.
This may be repeated.

**+s** defines a server on *FACE*.
*PREFIX* is a name prefix served by this server.
This may be repeated.

**-latency** enables latency measurements between client and server.
It requires client and server to run in the same process.

**-rtt** enables round trip time measurement on client.

**-add-delay** injects a delay before server answers an Interest.

**-nack=false** instructs the server to not respond to Interests it cannot serve, instead of responding with Nacks.

**-cnt** specifies duration between printing counters.

## Example

Emulate classical ndnping client:

```
sudo ndnping-dpdk EAL-ARGS -- -rtt +c dev://net_pcap0 /prefix/ping 100
```

Emulate classical ndnping server:

```
sudo ndnping-dpdk EAL-ARGS -- -nack=false +s dev://net_pcap0 /prefix/ping
```
