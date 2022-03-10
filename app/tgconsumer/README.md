# ndn-dpdk/app/tgconsumer

This package is the [traffic generator](../tg) consumer.
It may act as a [ndnping client](https://github.com/named-data/ndn-tools/blob/ndn-tools-22.02/tools/ping/README.md#ndnping-protocol).
It requires two threads, running `TgcTx_Run` and `TgcRx_Run` functions.

The consumer sends Interests and receives Data or Nacks.
It supports multiple configurable patterns:

* probability of selecting this pattern relative to other patterns
* Name prefix
* CanBePrefix flag
* MustBeFresh flag
* InterestLifetime value
* HopLimit value
* implicit digest
* relative sequence number

The consumer randomly selects a pattern and creates an Interest with the pattern settings.
The Interest name ends with a sequence number, which is a 64-bit integer encoded in binary format and native endianness.
Strictly speaking, this encoding violates the ndnping protocol, which requires the sequence number to be encoded in ASCII.
However, the current C++ `ndnpingserver` implementation can respond to such Interests.

The consumer maintains Interest, Data, Nack counters and collects Data round-trip time for each pattern.

## Implicit Digest

The consumer can generate Interests whose name contains an implicit digest component.
This is achieved by locally constructing a Data packet according to a Data template, and then computing its implicit digest.
Due to its performance overhead, patterns using this feature should be assigned lower weights.

When this feature is being used, the consumer contains a DPDK crypto device, and each pattern uses a queue pair.
Data packets for each pattern are prepared in bursts and enqueued into the crypto device submission queue.
Every time a pattern is selected, the consumer dequeues a Data packet from the crypto device completion queue, takes its computed implicit digest to construct an Interest, and then discards the Data packet.
In the rare case that the crypto device completion queue fails to return a Data packet, the consumer may randomly select another traffic pattern to use.

For the [producer](../tgproducer) to create Data packets that can satisfy those Interests, the producer's pattern should have a reply definition that has the same Data template.
Name suffix cannot be used in the Data template.

## Relative Sequence Number

Normally, each traffic pattern in the consumer has an independent sequence number that is incremented every time the pattern is selected.
If the `seqNumOffset` option is specified to a non-zero value, the traffic pattern would instead use a relative sequence number.

Every time this traffic pattern is selected:

1. The consumer reads the last sequence number of the previous pattern.
2. This sequence number is subtracted by `seqNumOffset`.
3. In the unlikely case that the computed sequence number is already requested recently, the sequence number is incremented.

This feature enables the consumer to generate Interests that can potentially be satisfied by cached Data in a forwarder, if the `seqNumOffset` setting is appropriate for the cache capacity and network latency.

Limitations of this feature:

* `seqNumOffset` cannot be specified on the first pattern.
* `seqNumOffset` cannot be specified together with implicit digest.
* Even if the previous pattern has implicit digest, Interests from this pattern do not have implicit digest.
* To reduce the probability of incremented sequence number (step 3), this pattern should be assigned a lower weight than the previous pattern.

## PIT token usage

The consumer encodes the following information in the PIT token field:

* when the Interest was sent,
* which pattern created the Interest,
* a "run number", so that replies to Interests from previous executions are not considered.

Having the above information inside the packet eliminates the need for a pending Interest table, allowing the consumer to operate more efficiently.
However, the consumer cannot detect network faults, such as unsolicited replies, duplicate replies, mismatched implicit digests.
