# ndn-dpdk/app/tgproducer

This package is the [traffic generator](../tg) producer.
It may act as a [ndnping server](https://github.com/named-data/ndn-tools/blob/ndn-tools-22.02/tools/ping/README.md#ndnping-protocol).
It requires at least one thread, running the `Tgp_Run` function.

The producer responds to every Interest with Data or Nack.
It supports multiple configurable patterns:

* Name prefix
* a list of possible reply definitions, each with a relative probability of being selected and one of:
  * Data template: Name suffix, FreshnessPeriod value, Content payload length
  * Nack reason
  * timeout/drop

Upon receiving an Interest, the producer finds a pattern whose name prefix is a prefix of the Interest name, and randomly picks a reply definition.
It then creates a Data or Nack packet according to the selected reply, unless the latter specifies a "timeout".
In case of a Data reply, the Data Name is the Interest name (except implicit digest) combined with the configured name suffix; if the name suffix is non-empty, the Interest needs to have the CanBePrefix flag.
An Interest that does not match any pattern is dropped.

The producer maintains counters for the number of processed Interests under each pattern and reply definition, and a counter for non-matching Interests.
