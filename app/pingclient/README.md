# ndn-dpdk/app/pingclient

This package is part of the [packet generator](../ping/).
It implements an **ndnping** client.
It runs `PingClientTx_Run` function in a *client-TX thread* ("CLIT" role) and `PingClientRx_Run` function in a *client-RX thread* ("CLIR" role) of the traffic generator.

The client sends Interests and receives Data or Nacks.
It supports multiple patterns that allow setting:

* probability weight of selecting this pattern relative to other patterns
* Name prefix
* CanBePrefix flag
* MustBeFresh flag
* InterestLifetime value
* HopLimit value
* override sequece number by subtracting previous pattern's sequence number with a fixed offset, allowing retrieving cached Data

The client randomly selects a pattern, and makes an Interest with the pattern settings.
The Interest name ends with a sequence number, which is a 64-bit number encoded in binary format and native endianness.
Strictly speaking, these sequence numbers violate the [ndnping Protocol](https://github.com/named-data/ndn-tools/blob/1fda67dc75692ccf0283a410f70db55686e2ff48/tools/ping/README.md#ndnping-protocol) that requires the sequence number to be encoded as ASCII.
However, the current C++ `ndnpingserver` implementation can respond to such Interests.

The client maintains Interest, Data, Nack counters and collects Data round-trip time under each pattern.

## PIT token usage

The client encodes these information in the PIT token field:

* when was the Interest sent
* which pattern created the Interest
* a "run number" so that replies to Interests from previous executions would not be accepted

Having these with the packet eliminates the need for a pending Interest table, allowing the client to operate more efficiently.
However, the client would not be able to detect network faults, such as unsolicited replies and double replies.
