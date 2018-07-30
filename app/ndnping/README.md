# ndn-dpdk/app/ndnping

This package implements a NDN package generator that doubles as **ndnping** client and server.

Unlike named-data.net's [ndnping](https://github.com/named-data/ndn-tools/tree/master/tools/ping) and [ndn-traffic-generator](https://github.com/named-data/ndn-traffic-generator), this implementation does not use a local forwarder, but directly sends and receives packets on a network interface.

## Client

The `Client` sends Interests and receives Data or Nacks.
It supports multiple *patterns* i.e. name prefixes.
The name suffix is a 64-bit sequence number encoded in binary format and in native endianness.
Other aspects of the Interest, such as InterestLifetime, are currently hard-coded.

Strictly speaking, binary sequence numbers violate the [ndnping Protocol](https://github.com/named-data/ndn-tools/blob/1fda67dc75692ccf0283a410f70db55686e2ff48/tools/ping/README.md#ndnping-protocol) that requires the sequence number to be encoded as ASCII.
However, the current C++ `ndnpingserver` implementation can respond to such Interests.

The client maintains Interest, Data, Nack counters under each pattern.
Optionally, the client can sample a subset of Interests and report round-trip time.

## Server

The `Server` responds to every Interest with Data or Nack.
It supports multiple *patterns* i.e. name prefixes.
Interests that fall under one of the prefixes are responded with Data.
Optionally, the server can respond Nack to Interests not matching any pattern.

The server maintains counters for the number of processed Interests under each pattern.

## Details on RTT Sampling

`Client.EnableRtt` method enables RTT sampling on a client.
If enabled, the C code will sample a subset of Interests.
For a sampled Interest, it records when the Interest is sent in `sampleTable`, and calculates RTT when the Data arrives.

`sampleFreq` argument controls sampling frequency.
If the last `sampleFreq` bits of a sequence number is all zeros, the client samples the Interest.
Because of this decision method, the number of patterns must be odd in order to obtain RTT samples on every pattern.

`sampleTableSize` argument sets the `sampleTable` to have `2 ^ sampleTableSize` entries.
The same entry will be reused after `2 ^ (sampleTableSize + sampleFreq)` Interests.
If an Interest is unanswered when its entry is due for reuse, the Interest is treated as timeout and its RTT not counted.
