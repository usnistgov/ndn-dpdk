# ndnping-dpdk

This program acts as [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) client or server on specified interfaces.
It can serve a traffic generator to benchmark a forwarder or a network.

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
sudo ndnping-dpdk EAL-ARGS -- -tasks="
---
- face:
    scheme: ether
    port: net_af_packet0
    local: "02:00:00:00:00:01"
    remote: "01:00:5e:00:17:aa"
  client:
    patterns:
      - prefix: /prefix/ping
        canbeprefix: false
        mustbefresh: true
    interval: 1ms
"
```

Emulate classical ndnping server:

```
sudo ndnping-dpdk EAL-ARGS -- -tasks="
---
- face:
    scheme: ether
    port: net_af_packet0
    local: "02:00:00:00:00:02"
    remote: "01:00:5e:00:17:aa"
  server:
    patterns:
      - prefix: /prefix/ping
        replies:
          - freshnessperiod: 1000ms
            payloadlen: 1024
    nack: true
"
```

## JSON-RPC API

This program provides a JSON-RPC API via [management RPC server](../../mgmt/).
It exports:

* [PingClient](../../mgmt/pingmgmt/): allow external control of defined ping clients in tasks.
* [Face](../../mgmt/facemgmt/): allow retrieval of face counters.
  Do not create/destroy/modify faces via RPC.
