# ndn-dpdk/ndni

This package implements NDN layer 2 and layer 3 packet representations for internal use in NDN-DPDK codebase.

Layer 2 implementation follows [**NDN Link Protocol v2 (NDNLPv2)** specification](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2), [revision 59](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2/63).
It supports indexed fragmentation, PIT token, network nack, and congestion mark features.

Layer 3 implementation follows [**NDN Packet Format** specification](https://docs.named-data.net/NDN-packet-spec/0.3/), [version 0.3](https://github.com/named-data/NDN-packet-spec/tree/023bac3047ac5a97677db7168e6dbf1108aec325).
The decoder supports TLV encoding evolvability in most situations.

## Low-Level TLV Functions

This package provides low-level pktmbuf and TLV functions including:

* Encoding and decoding of VAR-NUMBER used in TLV-TYPE and TLV-LENGTH.
* Encoding and decoding of NonNegativeInteger.
* Creating indirect pktmbuf.
* Linearizing pktmbuf.

## Name Representation

A name is represented as a buffer containing a sequence of name components, i.e. TLV-VALUE of the Name element.
TLV-LENGTH of the Name element cannot exceed `NameMaxLength`; this constant can be adjusted up to around 48KB, but it has implication in memory usage of table entries.

Two C types can represent a name:

* **LName** includes a pointer to the buffer of name components, and TLV-LENGTH of the Name element.
* **PName** additionally contains offsets of parsed name components.

All name components must be in continuous memory.
If this condition is not met, calling code must linearize the Name element's TLV-VALUE with `TlvDecoder_Linearize` function.
Having a linearized buffer, one can trivially construct an `LName`, or invoke `PName_Parse` function to construct a `PName`.
Neither type owns the name components buffer.

`PName` contains offsets of name components.
Internally, it only has space for the initial `PNameCachedComponents` name components.
However, its APIs allow unlimited number of name components: accessing a name component after `PNameCachedComponents` involves re-parsing and is inefficient.
`PName_ComputePrefixHash` function computes *SipHash* of the name or its prefix, which is useful in table implementation.

In Go, **ndn.Name** type should be used to represent a name.
To interact with C code, temporary allocate a `C.PName` via Go **ndni.PName** type.

## Packet Representation

In C, **Packet** type represents a L2 or L3 packet. `Packet*` is actually `struct rte_mbuf*`, with a `PacketPriv` placed at the private data area of the *direct* mbuf.
Within a `PacketPriv`:

* **LpL3** contains layer 3 fields in NDNLPv2 header, accessible via `Packet_GetLpL3Hdr`.
* **LpL2** contains layer 2 fields in NDNLPv2 header.
* **LpHeader** combines `LpL3` and `LpL2`, accessible via `Packet_GetLpHdr`.
* **PInterest** is a parsed Interest, accessible via `Packet_GetInterestHdr`.
  It must be used together with the mbuf containing the Interest packet.
* **PData** is a parsed Data, accessible via `Packet_GetDataHdr`.
  It must be used together with the mbuf containing the Data packet.
* **PNack** represents a parsed Nack, accessible via `Packet_GetNackHdr`.
  It overlays `LpL3` (where NackReason field is located) and `PInterest`.

`Packet_GetType` indicates what headers are currently accessible.
Attempting to access an inaccessible header type would result in assertion failure.

To receive and parse a packet, calling code should:

1. Ensure the direct mbuf has sufficiently large private data area for a `PacketPriv`.
2. Cast the mbuf to `Packet*` with `Packet_FromMbuf` function.
3. Invoke `Packet_Parse` function to parse the packet.
   NDNLPv2 headers are stripped from the mbuf during this step.
   Bare Interest/Data is considered valid LpPacket.
4. If `Packet_GetType` indicates the packet is a fragment, perform reassembly according to **LpL2**, and invoke `Packet_ParseL3` to parse network layer.
5. At this point, `PInterest`, `PData`, or `PNack` becomes available, and the mbuf only contains Interest or Data packet.
   Fragmented names are moved or copied into consecutive memory, allocated from the same mempool as the input mbuf.
   `LpL2` is overwritten but `LpL3` survives.

### Interest Decoding Details

`PInterest_Parse` function decodes an Interest.
If the Interest carries a *forwarding hint*, up to `PInterestMaxFwHints` names are recognized, and any remaining names are ignored.
The decoder only determines the length of each name, but does not parse at component level.
`PInterest_SelectFwHint` function activates a forwarding hint, parses the name into components on demand; only one name can be active at any time.

Although the packet format specifies Nonce as optional, it is required when an Interest is transmitted over network links.
Thus, `PInterest_Parse` requires Nonce to be present.
It stores the position of Nonce, InterestLifetime, and HopLimit elements in `nonceOffset` and `guiderSize` fields.
`Interest_ModifyGuiders` uses this information to modify these fields.

The decoder can accept unrecognized non-critical elements in most situations.
One exception is that, if there are too many unrecognized non-critical elements such that they inflate the distance between Nonce and HopLimit beyond 255 bytes, decoding will fail.
Also, `Interest_ModifyGuiders` does not preserve unrecognized non-critical elements between Nonce and HopLimit.

## Packet Encoding

There are limited support for packet encoding.

* **InterestTemplate** struct and related functions can encode Interest packets.
* **DataGen** struct and related functions can generate Data packets from a template without signing.
* `DataEnc_*` functions can generate Data packets with payload without signing.
* `Nack_FromInterest` turns an Interest packet into a Nack packet in-place.
