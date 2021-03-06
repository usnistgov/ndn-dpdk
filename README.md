# NDN-DPDK: High-Speed Named Data Networking Forwarder

NDN-DPDK is a set of high-speed [Named Data Networking (NDN)](https://named-data.net/) programs developed with the [Data Plane Development Kit (DPDK)](https://www.dpdk.org/).
Included are a network forwarder and a traffic generator.

![NDN-DPDK logo](docs/NDN-DPDK-logo.svg)

This software is developed at the [Advanced Network Technologies Division](https://www.nist.gov/itl/antd) of the [National Institute of Standards and Technology](https://www.nist.gov/).
It is in pre-release stage and will continue to be updated.

## Documentation

* [NDN-DPDK installation guide](docs/INSTALL.md)
* [NDN-DPDK Docker container](docs/Docker.md)
* [NDN-DPDK forwarder activation and usage](docs/forwarder.md)
* [hardware known to work with NDN-DPDK](docs/hardware.md)
* [NDN-DPDK publications and presentations](docs/publication.md)
* [godoc](https://pkg.go.dev/github.com/usnistgov/ndn-dpdk)

If you use NDN-DPDK in your research, please cite the [NDN-DPDK paper](docs/publication.md) instead of this GitHub repository.

## Features

Packet encoding and decoding

* Interest and Data
  * [v0.3](https://named-data.net/doc/NDN-packet-spec/0.3/) format only
  * TLV evolvability: no
  * forwarding hint: yes
* [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2)
  * fragmentation and reassembly: yes
  * Nack: yes
  * PIT token: yes, only support 8-octet length
  * congestion mark: yes
  * link layer reliability: no

Transports

* DPDK-based high-speed transports: Ethernet, VLAN, UDP, VXLAN
  * Ethernet adapter must be dedicated to DPDK
* socket-based transports via kernel: UDP, TCP
* local application transports: memif, Unix sockets

Forwarding plane

* multi-threaded architecture
* forwarding strategies: eBPF programs
* FIB: includes strategy choice and statistics
* PIT-CS Composite Table (PCCT): includes PIT and CS

Management

* GraphQL endpoint: yes
* configuration file: none
* routing: no
  * [Multiverse](https://github.com/multiverse-nms) can provide centralized routing

## Code Organization

* [ndn](ndn): NDN library in pure Go.
* [mk](mk): build helper scripts.
* [csrc](csrc): C source code.
* [js](js): TypeScript source code.
* [core](core): common shared code.
* [dpdk](dpdk): Go bindings for DPDK and SPDK.
* [ndni](ndni): NDN packet representation for internal use.
* [iface](iface): network interfaces.
* [container](container): data structures.
* [strategy](strategy): forwarding strategy BPF programs.
* [app](app): applications, including the forwarder dataplane.
* [cmd](cmd): executables.

These is a `README.md` file in most directories of this codebase that explains the relevant module.
