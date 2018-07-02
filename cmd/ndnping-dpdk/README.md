# ndnping-dpdk

This program acts as [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) client or server on specified interfaces.

## Usage

```
sudo ndnping-dpdk EAL-ARGS -- \
  [-initcfg=INITCFG] \
  [-latency] [-rtt] \
  [-add-delay=DURATION] [-nack=false] [-suffix=/NAME] [-payload-len=SIZE] \
  [-cnt DURATION] \
  +c FACE INTERVAL PREFIX PCT PREFIX PCT \
  +s FACE PREFIX PREFIX
```

**-initcfg** accepts an initialization configuration object in YAML format.
This program recognizes *mempool* section only.

**+c** defines a client on *FACE*.
*INTERVAL* is the interval between two Interests; since the client sends Interests in bursts, this will be transformed into a burst interval where the average Interest interval matches this specified interval; zero means "as fast as possible".
*PREFIX* is a name prefix for Interests; it is recommended to use `ping` as the last name component.
*PCT* is the percentage of traffic under this prefix; this sub-option is not implemented.
This may be repeated.

**+s** defines a server on *FACE*.
*PREFIX* is a name prefix served by this server.
This may be repeated.

**-latency** enables latency measurements between client and server.
It requires client and server to run in the same process.
This feature is not implemented.

**-rtt** enables round trip time measurement on client.
This option is not implemented: RTT measurement is always enabled on pattern 0, and disabled on other patterns.

**-add-delay** injects a delay before server answers an Interest.
This feature is not implemented.

**-nack=false** instructs the server to not respond to Interests it cannot serve, instead of responding with Nacks.

**-suffix** appends a suffix to Data names from the server.

**-payload-len** specifies length of Content in server's Data packets.

**-cnt** specifies duration between printing counters.

## Example

Emulate classical ndnping client:

```
sudo ndnping-dpdk EAL-ARGS -- -rtt +c dev://net_pcap0 1ms /prefix/ping 100
```

Emulate classical ndnping server:

```
sudo ndnping-dpdk EAL-ARGS -- -nack=false +s dev://net_pcap0 /prefix/ping
```
