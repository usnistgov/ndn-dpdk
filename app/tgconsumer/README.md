# ndn-dpdk/app/tgconsumer

This package is the [traffic generator](../tg) consumer.
It may act as a [ndnping client](https://github.com/named-data/ndn-tools/blob/ndn-tools-0.7.1/tools/ping/README.md#ndnping-protocol).
It requires two threads, running `TgcTx_Run` and `TgcRx_Run` functions.

The consumer sends Interests and receives Data or Nacks.
It supports multiple configurable patterns:

* probability of selecting this pattern relative to other patterns
* Name prefix
* CanBePrefix flag
* MustBeFresh flag
* InterestLifetime value
* HopLimit value
* override the sequence number by subtracting a fixed offset from the previous pattern's sequence number, thus allowing to retrieve cached Data
* implicit digest

The consumer randomly selects a pattern and creates an Interest with the pattern settings.
The Interest name ends with a sequence number, which is a 64-bit integer encoded in binary format and native endianness.
Strictly speaking, this encoding violates the ndnping protocol, that requires the sequence number to be encoded in ASCII.
However, the current C++ `ndnpingserver` implementation can respond to such Interests.

The consumer maintains Interest, Data, Nack counters and collects Data round-trip time for each pattern.

## Implicit Digest

The consumer can generate Interests whose name contains an implicit digest component.
This is achieved by locally constructing a Data packet according to a Data template, and then computing its implicit digest using a DPDK crypto device.
Due to its performance overhead, patterns using this feature should be assigned lower weights.

For the [producer](../tgproducer) to create Data packets that can satisfy those Interests, the producer's pattern should have a reply definition that has the same Data template.
Name suffix cannot be used in the Data template.

## PIT token usage

The consumer encodes the following information in the PIT token field:

* when the Interest was sent,
* which pattern created the Interest,
* a "run number", so that replies to Interests from previous executions are not considered.

Having the above information inside the packet eliminates the need for a pending Interest table, allowing the consumer to operate more efficiently.
However, the consumer cannot detect network faults, such as unsolicited replies, duplicate replies, mismatched implicit digests.
