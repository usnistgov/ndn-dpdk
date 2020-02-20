# ndn-dpdk/cmd/benchmark

This directory contains forwarder benchmark scripts.
They control an [`ndnping-dpdk`](../ndnping-dpdk/) traffic generator via [pingmgmt commands](../../mgmt/pingmgmt/).

## MSI Benchmark

`msi.ts` attempts to find **minimum sustained interval** (MSI).
It minimizes the Interest sending interval, such that Interest satisfaction ratio stays near 100% within a period of time.
Measured MSI can be used to calculate the throughput of a forwarder or a network.

`msibench.ts` repeats MSI measurement to reach a specified desired uncertainty.

To use them, start the traffic generator and rest of the network:

```
sudo MGMT=tcp://127.0.0.1:6345 ndnping-dpdk --vdev net_af_packet1,iface=eth1 -- \
  -cnt 1s -initcfg @init-config.yaml -tasks @ndnping.yaml
```

* It's necessary to make JSON-RPC server listen on TCP via [`MGMT` environment variable](../../mgmt/), because JSON-RPC client does not support Unix sockets.
* `-tasks` object must contain one or more clients.
* The interval specified in `-tasks` object is ignored during benchmark.
* Ensure you are getting Data replies before proceeding.

Then, launch a benchmark script:

```
DEBUG=* node build/cmd/benchmark/msi --IntervalMin 1000 --IntervalMax 5000 --IntervalStep 1

DEBUG=* node build/cmd/benchmark/msibench --IntervalMin 1000 --IntervalMax 5000 --IntervalStep 1 --DesiredUncertainty 100
```

* `--IntervalMin` and `--IntervalMax` options indicate the range (in nanoseconds) to search MSI within.
  If MSI is out of this range, the benchmark will fail.
* `--IntervalStep` option indicates how accurate each single MSI should be.
* See source code comments for explanation about many other options.
* Results are written in JSON format on stdout, one JSON object per line.
* Set `DEBUG=*` environment variable to enable debug logs on stderr.

## Fetch Benchmark

`fetchbench.ts` runs [fetch](../../app/fetch/) repeatedly until **goodput** reaches a specified desired uncertainty.
Goodput is defined as the number of successfully retrievd Data segments per second, excluding duplicate Data segments caused by Interest retransmissions.
Its unit is "Data packets per second".

To use this script, first start the traffic generator, with a fetcher defined in `-tasks` object.
Then, launch the benchmark script:

```
DEBUG=* node build/cmd/benchmark/fetchbench --Index 0 --NamePrefix /8=producer/8=prefix --NameCount 6 --DesiredUncertainty 30000
```

* `--Index` option indicates which "task" has the fetcher.
* `--NamePrefix` is the name prefix.
* `--NameCount` is the number of distinct name prefixes.
* Interest names have format *NamePrefix*/*i*/*random*/*segment-number*, where *i* is between 0 and NameCount.
