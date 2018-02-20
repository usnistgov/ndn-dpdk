# ndn-dpdk/ndn

This package implements NDN layer 2 and layer 3 packet representations.

Layer 2 implementation follows **NDN Link Protocol v2 (NDNLPv2)** specification, [revision 27](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2/27).
It supports index fragmentation, network nack, and congestion mark features.
As a protocol extension, it supports [PIT token](https://redmine.named-data.net/issues/4432) field.

Layer 3 implementation follows **NDN Packet Format** specification, [version 0.3 draft 4441,24](https://gerrit.named-data.net/#/c/4441/24).
However, it does not support TLV encoding evolvability: encountering an unrecognized or out-of-order TLV element would cause the packet to be treated as invalid, regardless of whether its TLV-TYPE is critical or non-critical.

## Low-Level TLV Functions

This package provides low-level TLV functions including:

* Encoding and decoding of variable size numbers used in TLV-TYPE and TLV-LENGTH.
* TLV element representation.

## Name Representation

A name is represented as a buffer containing a sequence of name components, i.e. TLV-VALUE of the Name element.
TLV-LENGTH of the Name element cannot exceed `NAME_MAX_LENGTH`; this constant can be adjusted up to around 48KB, but it has implication in memory usage of table entries.

Three C types can represent a name:

* **LName** includes a pointer to the buffer of name components, and TLV-LENGTH of the Name element.
* **PName** contains offsets of parsed name components, but does not have a pointer to the buffer.
* **Name** combines a pointer to the buffer with a `PName`.

It is required that all name components are in a consecutive memory buffer. If this condition is not met, calling code must linearize the Name element's TLV-VALUE with `TlvElement_LinearizeValue` function.
Having a linearized buffer, one can trivially construct an `LName`, or invoke `PName_Parse` function to construct a `PName`.
`Name` is for enclosing into a larger struct; `Name*` can be cast to `LName*`.
None of these types own the name components buffer.

`PName` contains offsets of name components.
Internally, it only has space for the initial `PNAME_N_CACHED_COMPS` name components.
However, its APIs allow unlimited number of name components: accessing a name component after `PNAME_N_CACHED_COMPS` involves re-parsing and is inefficient.
`PName_GetCompBegin` and `PName_GetCompEnd` functions provide the boundary of each name component.
`PName_ComputePrefixHash` function computes *SipHash* of the name or its prefix, which is useful in table implementation.
All `PName` APIs require a pointer to the name components buffer that the `PName` was parsed from.

In Go, **Name** type represents a name.
It contains a name components buffer and a `C.PName` instance.
There are also functions to convert between `Name` and the corresponding NDN URI representation.

## Packet Representation

In C, **Packet** type represents a L2 or L3 packet. `Packet*` is actually `struct rte_mbuf*`, with a `PacketPriv` placed at the private data area of the *direct* mbuf.
Within a `PacketPriv`:

* **LpL3** contains layer 3 fields in NDNLPv2 header, accessible via `Packet_GetLpL3Hdr`.
* **LpL2** contains layer 2 fields in NDNLPv2 header.
* **LpHeader** combines `LpL3` and `LpL2`, accessible via `Packet_GetLpHdr`.
* **PInterest** is a parsed Interest, accessible via `Packet_GetInterestHdr`. It must be used together with the mbuf containing the Interest packet.
* **PData** is a parsed Data, accessible via `Packet_GetDataHdr`. It must be used together with the mbuf containing the Data packet.
* **PNack** represents a parsed Nack, accessible via `Packet_GetNackHdr`. It overlays `LpL3` (where NackReason field is located) and `PInterest`.

`Packet_GetL2PktType` and `Packet_GetL3PktType` indicate what headers are currently accessible.
Attempting to access an inaccessible header type would result in assertion failure.

To receive and parse a packet, calling code should:

1. Ensure the direct mbuf has sufficiently large private data area for a `PacketPriv` (this size is exposed as `SizeofPacketPriv()` in Go).
2. Cast the mbuf to `Packet*` with `Packet_FromMbuf` function.
3. Invoke `Packet_ParseL2` function to parse NDNLPv2 headers into `LpHeader`. NDNLPv2 headers are stripped from the mbuf during this step. Bare Interest/Data is considered valid LpPacket, and also makes `LpHeader` available.
4. Perform reassembly if necessary.
5. Invoke `Packet_ParseL3` function to parse Interest or Data into `PInterest`, `PData`, or `PNack`. During this step, fragmented names are moved or copied into consecutive memory. `LpHeader` is overwritten but `LpL3` survives.

### Interest Decoding Details

`PInterest_FromPacket` function decodes an Interest.

If the Interest carries a *forwarding hint*, up to `INTEREST_MAX_FHS` delegations are recognized, and any remaining delegations are ignored.
`PInterest.fh` array stores the recognized delegation names as `LName`; this implies that the decoder only determines the length of each name, but does not parse at component level.
`PInterest_ParseFh` function can parse a delegation name into components on demand, but only one delegation name can be stored as as `PName` in a `PInterest`.

If the Interest carries the *HopLimit* field, it is automatically decremented in place.

The decoder stores the position of Nonce and InterestLifetime fields as `PInterest.guiderLoc`. It can be used to insert a missing Nonce field, or to modify InterestLifetime.

## Packet Encoding

There are limited support for packet encoding.

* **InterestTemplate** struct and related functions encode an Interest. Its Go binding does not support forwarding hint and Parameters field.
* `EncodeData` functions make a Data with given name and payload. It will attach an invalid HMAC signature.
* `MakeNack` turns an Interest into a Nack in-place. It does not have Go binding.
