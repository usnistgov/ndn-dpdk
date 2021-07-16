# NDNgo: Named Data Networking in Go

**NDNgo** is a minimal [Named Data Networking](https://named-data.net/) library compatible with the NDN-DPDK forwarder.
The main purpose of this library is for implementing unit tests and management functionality of NDN-DPDK.
It also serves as a demonstration on how to create a library compatible with NDN-DPDK.

NDNgo does not depend on Cgo, and can be used in external projects via Go Modules.
It is intended to be cross-platform, and part of the library can be compiled for WebAssembly via [TinyGo](https://tinygo.org/) compiler.
However, this is not a high performance library, and there is no API stability guarantees.

![NDNgo logo](../docs/NDNgo-logo.svg)

## Features

Packet encoding and decoding

* General purpose TLV codec (in [package tlv](tlv))
* Interest and Data: [v0.3](https://named-data.net/doc/NDN-packet-spec/0.3/) format only
  * TLV evolvability: yes
  * Forwarding hint: yes
  * Signed Interest: basic support
* [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2)
  * Fragmentation and reassembly: partial
  * Nack: yes
  * PIT token: yes
  * Congestion mark: yes
  * Link layer reliability: no
* Naming Convention: TLV-TYPE numbers only

Transports

* Unix stream, UDP unicast, TCP (in [package sockettransport](sockettransport))
* Ethernet via [GoPacket library](https://github.com/google/gopacket) (in [package packettransport](packettransport))
* Shared memory with local NDN-DPDK forwarder via [memif](https://pkg.go.dev/github.com/FDio/vpp/extras/gomemif/memif) (in [package memiftransport](memiftransport))

KeyChain

* Encryption: no
* Signing algorithms
  * SHA256: yes
  * ECDSA: yes
  * RSA: yes
  * HMAC-SHA256: no
  * [Null](https://redmine.named-data.net/projects/ndn-tlv/wiki/NullSignature): yes
* [NDN certificates](https://named-data.net/doc/ndn-cxx/0.7.1/specs/certificate-format.html): decoding
* Key persistence: no
* Trust schema: no

Application layer services

* Endpoint: yes
* Segmented object: consumer and producer (in [package segmented](segmented))

## Getting Started

The best places to get started are:

* `Consume` function in [package endpoint](endpoint): express an Interest and wait for response, with automatic retransmissions and Data verification.
* `Produce` function in [package endpoint](endpoint): start a producer, with automatic Data signing.
* `l3.Face` type in [package l3](l3): network layer face abstraction, for low-level programming.

Examples are in [command ndndpdk-godemo](../cmd/ndndpdk-godemo).
