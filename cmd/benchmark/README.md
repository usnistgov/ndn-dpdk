# ndn-dpdk/cmd/benchmark

This directory contains forwarder benchmark tools.
These tools expect an [`ndnping-dpdk`](../ndnping-dpdk/) traffic generator with one or more clients, and control the generator via [pingmgmt commands](../../mgmt/pingmgmt/).

`msi.ts` attempts to find **minimum sustained interval** (MSI).
It minimizes the Interest sending interval, such that Interest satisfaction ratio stays near 100% within a period of time.
Measured MSI can be used to calculate the throughput of a forwarder or a network.

`msibench.ts` repeats MSI measurement to reach a specified desired uncertainty.

## Instructions

First, start the traffic generator and rest of the network:

```
sudo MGMT=tcp://127.0.0.1:6345 ndnping-dpdk --vdev net_af_packet1,iface=eth1 -- \
  -cnt 1s -initcfg @init-config.yaml -tasks @ndnping.yaml
```

* It's necessary to make JSON-RPC server listen on TCP via [`MGMT` environment variable](../../mgmt/), because JSON-RPC client does not support Unix sockets.
* The interval specified in `-tasks` object is ignored during benchmark.
* Ensure you are getting Data replies before proceeding.

Then, launch a benchmark tool:

```
DEBUG=* node build/cmd/benchmark/msi --IntervalMin 1000 --IntervalMax 5000 --IntervalStep 1

DEBUG=* node build/cmd/benchmark/msibench --IntervalMin 1000 --IntervalMax 5000 --IntervalStep 1 --DesiredUncertainty 100
```

* `--IntervalMin` and `--IntervalMax` options indicate the range (in nanoseconds) to search MSI within.
  If MSI is out of this range, the benchmark will fail.
* `--IntervalStep` option indicate how accurate each single MSI should be.
* See source code comments for explanation about many other options.
* Results are written in JSON format on stdout, one JSON object per line.
* Set `DEBUG=*` environment variable to enable debug logs on stderr.
