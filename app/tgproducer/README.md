# ndn-dpdk/app/tgproducer

This package is the [traffic generator](../tg) producer.
It may act as a [ndnping server](https://github.com/named-data/ndn-tools/blob/ndn-tools-0.7.1/tools/ping/README.md#ndnping-protocol).
It requires one thread, running the `Tgp_Run` function.

The producer responds to every Interest with Data or Nack.
It supports multiple configurable patterns:

* Name prefix
* a list of possible response definitions, each with a relative probability of being selected and one of:
  * Data template: Name suffix, FreshnessPeriod value, Content payload length
  * Nack reason
  * timeout/drop

Upon receiving an Interest, the producer finds a pattern whose name prefix is a prefix of the Interest name, and randomly picks a response definition.
It then creates a Data or Nack packet according to the selected definition, unless the latter specifies a "timeout".
In case of a Data reply, the Data Name is the Interest name combined with the configured name suffix; if the name suffix is non-empty, the Interest needs to have the CanBePrefix flag.
If no pattern matches the Interest, the producer can optionally reply with a Nack.

The producer maintains counters for the number of processed Interests under each pattern and response definition, and a counter for non-matching Interests.

The producer can emulate a minimum processing delay for testing purposes.
If this is enabled, the reply to each packet will not leave the producer thread until the minimum processing delay has elapsed since the packet arrived at the input thread.
For efficiency reasons, this delay is applied per burst of packets, using the timestamp on the last packet of each burst.
