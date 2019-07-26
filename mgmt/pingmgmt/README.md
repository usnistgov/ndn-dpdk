# ndn-dpdk/mgmt/pingmgmt

This package allows controlling [ndnping](../../app/ndnping/) process via RPC.

## PingClient

Ping clients must be defined in `ndnping.TaskConfig` when starting ndnping application.
They are initially in "running" state, but can be stopped and (re-)started using these APIs.
The APIs are designed to facilitate throughput benchmarks, so that it has limited functionality, and does not support creating new clients or changing traffic patterns.

**PingClient.List** listing defined ping clients.

**PingClient.Start** starts an stopped ping client.
It optionally allows changing Interest sending interval and clearing counters.

**PingClient.Stop** stops a running ping client.

**PingClient.ReadCounters** reads counters from a running or stopped ping client.
