# ndnping-dpdk

This program acts as [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) client or server on specified interfaces.

## Usage

```
sudo ndnping-dpdk EAL-ARGS -- [-initcfg=INITCFG] [-tasks=TASKS] [-cnt DURATION]
```

**-initcfg** accepts an initialization configuration object in YAML format.
This program recognizes *mempool* section only.

**-tasks** accepts a task description object in YAML format.

**-cnt** specifies duration between printing counters.

## Example

Emulate classical ndnping client:

```
sudo ndnping-dpdk EAL-ARGS -- -tasks="[{face:{remote:'dev://net_pcap0'},client:{patterns:[{prefix:"/prefix/ping"}],interval:1ms}}]"
```

Emulate classical ndnping server:

```
sudo ndnping-dpdk EAL-ARGS -- -tasks="[{face:{remote:'dev://net_pcap0'},server:{patterns:[{prefix:"/prefix/ping"}],nack:false}}]"
```
