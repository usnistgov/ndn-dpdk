# ndn-dpdk/app/ping

This package implements a NDN packet generator.

Unlike named-data.net's [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) and [ndn-traffic-generator](https://github.com/named-data/ndn-traffic-generator), this implementation does not use a local forwarder, but directly sends and receives packets on a network interface.

This packet generator has up to five threads for each face:

* The *input thread* ("RX" role) runs an **iface.RxLoop** with [InputDemux3](../inputdemux).
  It dispatches Data and Nacks to client-RX thread, and dispatches Interests to server thread.
* Either the *client-TX thread* and *client-RX thread* run a [ping client](../pingclient/), or a *fetcher thread* runs a [fetcher](../fetch/).
* The *server thread* runs a [ping server](../pingserver/).
* The *output thread* ("TX" role) runs an **iface.TxLoop** that transmits Interests, Data, and Nacks created by client-RX thread or server thread.

```
      /--client0-RX
      |             client0-TX--\
      |                         |
input-+---------fetch1----------+-output
      |                         |
      +---------server0---------+
      \---------server1---------/
```
