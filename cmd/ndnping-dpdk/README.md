# ndnping-dpdk

This program acts as an [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) client or server on the specified interfaces.
It can serve as a traffic generator to benchmark a forwarder or a network.

## Usage

```sh
sudo ndnping-dpdk EAL-ARGS -- [-initcfg=INITCFG] [-tasks=TASKS] [-cnt DURATION]
```

**-initcfg** accepts an initialization configuration object in YAML format.
This program recognizes the `Mempool` section only.
See [here](../../docs/init-config.sample.yaml) for an example.

**-tasks** accepts a task description object in YAML format.
See [here](../../docs/ndnping.sample.yaml) and below for examples.

**-cnt** specifies the time interval between printing counters.
The argument value must be a duration string acceptable to Go's [time.ParseDuration](https://golang.org/pkg/time/#ParseDuration).

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

This program provides a JSON-RPC API via the [management RPC server](../../mgmt).
It exports the following interfaces:

* [PingClient](../../mgmt/pingmgmt): allows external control of the ping clients defined in *-tasks=*.
* [Face](../../mgmt/facemgmt): allows retrieving the face counters.
  Do not create/destroy/modify faces via this RPC interface.
