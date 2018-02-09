# ndn-dpdk/ndn

This package implements NDN packet representations.

## Packet Decoding and Representation

`InterestPkt`, `DataPkt`, and `LpPkt` represent Interest, Data, and NDNLP packets, respectively.
They can be decoded from `TlvDecodePos` that is really a `MbufLoc`.

These types are designed to represent a decoded packet.
They contain pointers into the original packet, and thus must be used together with the input mbufs.
They cannot be used to modify the packet.

## Encoding

There are limited support for packet encoding:

* Parse `ndn:` URI into Name (bytes).
* Encode NDNLP packet (bytes).
* Turn Interest into Nack (mbuf, in place).
