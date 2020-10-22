# ndn-dpdk/app/tg

This package implements an NDN traffic generator.

Unlike named-data.net's [ndnping](https://github.com/named-data/ndn-tools/tree/ndn-tools-0.7.1/tools/ping) and [ndn-traffic-generator](https://github.com/named-data/ndn-traffic-generator) programs, this implementation does not use a local forwarder, but directly sends and receives packets on a network interface.

This traffic generator has up to five threads for each face:

* The *input thread* ("RX" role) runs an **iface.RxLoop** that dispatches Data/Nacks to the consumer and dispatches Interests to the producer.
* The *output thread* ("TX" role) runs an **iface.TxLoop** that transmits Interests, Data, and Nacks created by the client-RX and server threads.
* Either:
  * two *consumer threads* ("CONSUMER" role) run a [traffic generator consumer](../tgconsumer); or
  * one *consumer thread* ("CONSUMER" role) runs a [fetcher](../fetch).
* The *producer thread* ("PRODUCER" role) runs a [traffic generator producer](../tgproducer).

```
      /--consumer0-RX
      |           consumer0-TX--\
      |                         |
input-+---------fetch1----------+-output
      |                         |
      +--------producer0--------+
      \--------producer1--------/
```
