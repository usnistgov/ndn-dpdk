# ndn-dpdk/mgmt/pingmgmt

This package allows controlling [ndnping](../../app/ping/) process via RPC.
The APIs are designed to facilitate throughput benchmarks, so that they have limited functionality.

## PingClient

Ping clients must be defined in `ping.TaskConfig` when starting ndnping application.
They are initially in "running" state, but can be stopped and (re-)started using these APIs.

**PingClient.List** lists defined ping clients.

**PingClient.Start** starts an stopped ping client.
It optionally allows changing Interest sending interval and clearing counters.

**PingClient.Stop** stops a running ping client.

**PingClient.ReadCounters** reads counters from a running or stopped ping client.

## Fetch

Fetchers must be defined in `ndnping.TaskConfig` when starting ndnping application.
The fetcher must be stopped (`ping.FetchConfig` should have `Name` set to nil), and benchmark can be executed only once on a fetcher.

**Fetch.List** lists defined fetchers.

**Fetch.Benchmark** executes a fetcher in benchmark mode.
