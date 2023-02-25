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
* Interest and Data: [v0.3](https://docs.named-data.net/NDN-packet-spec/0.3/) format only
  * TLV evolvability: yes
  * Forwarding hint: yes
  * Signed Interest: basic support
* [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2)
  * Fragmentation and reassembly: partial
  * Nack: yes
  * PIT token: yes
  * Congestion mark: yes
  * Link layer reliability: no
* Naming Convention: [rev3 format](https://named-data.net/publications/techreports/ndn-tr-22-3-ndn-memo-naming-conventions/) ([TLV-TYPE numbers](https://redmine.named-data.net/projects/ndn-tlv/wiki/NameComponentType/29))

Transports

* Unix stream, UDP unicast, TCP (in [package sockettransport](sockettransport))
* Ethernet via [GoPacket library](https://github.com/google/gopacket) (in [package packettransport](packettransport))
* Shared memory with local NDN-DPDK forwarder via [memif](https://pkg.go.dev/github.com/FDio/vpp/extras/gomemif/memif) (in [package memiftransport](memiftransport))
* WebSocket for WebAssembly (in [package wasmtransport](wasmtransport))

KeyChain

* Encryption: no
* Signing algorithms
  * SHA256: yes
  * ECDSA: yes
  * RSA: yes
  * HMAC-SHA256: no
  * Ed25519: proof of concept only
  * Null: yes
* [NDN certificates](https://docs.named-data.net/NDN-packet-spec/0.3/certificate.html): basic support
  * [SafeBag](https://docs.named-data.net/ndn-cxx/0.8.1/specs/safe-bag.html): import and export
* Persistent key and certificate storage: no
* Trust schema: no

Application layer services

* Endpoint: yes
* Segmented object: consumer and producer (in [package segmented](segmented))
* [Realtime Data Retrieval (RDR)](https://redmine.named-data.net/projects/ndn-tlv/wiki/RDR): metadata structure (in [package rdr](rdr))

Management integration:

* Connecting to NDN-DPDK: yes (in [package gqlmgmt](mgmt/gqlmgmt))
* Connecting to NFD and YaNFD: yes (in [package nfdmgmt](mgmt/nfdmgmt))
* [NDN-FCH 2021](https://github.com/11th-ndn-hackathon/ndn-fch): client (in [package fch](fch))

## Getting Started

The best places to get started are:

* `Consume` function in [package endpoint](endpoint): express an Interest and wait for response, with automatic retransmissions and Data verification.
* `Produce` function in [package endpoint](endpoint): start a producer, with automatic Data signing.
* `l3.Face` type in [package l3](l3): network layer face abstraction, for low-level programming.

Examples are in [command ndndpdk-godemo](../cmd/ndndpdk-godemo).
