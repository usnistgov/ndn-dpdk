# ndnping-dpdk

This program acts as [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) client or server on specified interfaces.

## Usage

```
sudo ndnping-dpdk EAL-ARGS -- [-initcfg=INITCFG] [-tasks=TASKS] [-cnt DURATION] [-throughput-benchmark=CONFIG]
```

**-initcfg** accepts an initialization configuration object in YAML format.
This program recognizes *mempool* section only.

**-tasks** accepts a task description object in YAML format.

**-cnt** specifies duration between printing counters.

**-throughput-benchmark** accepts a throughput benchmark config object in YAML format.

## Example

Emulate classical ndnping client:

```
sudo ndnping-dpdk EAL-ARGS -- -tasks="
---
- face:
    remote: ether://01-00-5E-00-17-AA@net_af_packet0:0
    remote: ether://02-00-00-00-00-01@net_af_packet0:0
  client:
    patterns:
      - prefix: /prefix/ping
    interval: 1ms
"
```

Emulate classical ndnping server:

```
sudo ndnping-dpdk EAL-ARGS -- -tasks="
---
- face:
    remote: ether://01-00-5E-00-17-AA@net_af_packet0:0
    remote: ether://02-00-00-00-00-02@net_af_packet0:0
  server:
    patterns:
      - prefix: /prefix/ping
    nack: true
"
```

## Throughput Benchmark Mode

When **-throughput-benchmark** command line option is given, the program enters throughput benchmark mode.
To use this mode, the task description object must have at least one task, and the first task must contains a client, which will be taken over by throughput benchmark module.
It is recommended to disable periodical counter printing (`-cnt 0`) when using this mode.
To watch progress, enable logging with `LOG_ThroughputBenchmark=V` environ.

Throughput benchmark module attempts to find **minimum sustained interval** (MSI).
It minimizes the Interest sending interval, such that Interest satisfaction ratio stays near 100% within a period of time.
Measured MSI can be used to calculate the throughput of a forwarder or a network.

Example:

```
sudo LOG_ThroughputBenchmark=V ndnping-dpdk EAL-ARGS -- -cnt=0 -tasks="
---
- face:
    remote: ether://01-00-5E-00-17-AA@net_af_packet0:0
    remote: ether://02-00-00-00-00-01@net_af_packet0:0
  client:
    patterns:
      - prefix: /prefix/ping
    interval: 1ms
" -throughput-benchmark="
---
intervalmin: 500ns
intervalmax: 2500ns
intervalstep: 1ns

txcount: 24000000
txdurationmin: 15s
txdurationmax: 60s

warmuptime: 5s
cooldowntime: 2s
readcountersfreq: 100ms

satisfythreshold: 0.999
retestthreshold: 0.950
retestcount: 1
"
```
