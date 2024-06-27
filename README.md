# NDN-DPDK: High-Speed Named Data Networking Forwarder

NDN-DPDK is a set of high-speed [Named Data Networking (NDN)](https://named-data.net/) programs developed with the [Data Plane Development Kit (DPDK)](https://www.dpdk.org/).
Included are a network forwarder, a traffic generator, and a file server.

![NDN-DPDK logo](docs/NDN-DPDK-logo.svg)

This software is developed at the [Smart Connected Systems Division](https://www.nist.gov/ctl/smart-connected-systems-division) of the [National Institute of Standards and Technology](https://www.nist.gov/).
It is in beta stage and will continue to be updated.

Acknowledgement: NDN-DPDK development and testing make use of public testbeds, including [FABRIC](https://whatisfabric.net), [Cloudlab](https://www.cloudlab.us), [Emulab](https://www.emulab.net), [Virtual Wall](https://doc.ilabt.imec.be/ilabt/virtualwall/index.html), [Grid'5000](https://www.grid5000.fr).

## Documentation

* [Installation guide](docs/INSTALL.md)
* [Docker container](docs/Docker.md)
* [Forwarder activation and usage](docs/forwarder.md)
* [Traffic generator activation and usage](docs/trafficgen.md)
* [File server activation and usage](docs/fileserver.md)
* [Hardware known to work](docs/hardware.md)
* [Face creation](docs/face.md)
* [Performance tuning](docs/tuning.md)
* [Interoperability with other NDN software](docs/interop)
* [Related publications and presentations](docs/publication.md)
* [Doxygen reference](https://ndn-dpdk.ndn.today/doxygen/)
* [Go reference](https://pkg.go.dev/github.com/usnistgov/ndn-dpdk)

If you use NDN-DPDK in your research, please cite the [NDN-DPDK paper](docs/publication.md) instead of this GitHub repository.

## Features

Packet encoding and decoding

* Interest and Data: [v0.3](https://docs.named-data.net/NDN-packet-spec/0.3/) format only
  * TLV evolvability: yes
  * Forwarding hint: yes
* [NDNLPv2](https://redmine.named-data.net/projects/nfd/wiki/NDNLPv2)
  * Fragmentation and reassembly: yes
  * Nack: yes
  * PIT token: yes
  * Congestion mark: yes
  * Link layer reliability: no

Transports

* Ethernet-based transports via DPDK: Ethernet, VLAN, UDP, VXLAN, GTP-U
* Socket-based transports via kernel: UDP, TCP
* Local application transports: memif, Unix sockets

Forwarding plane

* Multi-threaded architecture
* Forwarding strategies: eBPF programs
* FIB: includes strategy choice and statistics
* PIT-CS Composite Table (PCCT): includes PIT and CS

Management

* GraphQL endpoint: HTTP POST, WebSocket "graphql-transport-ws", WebSocket "graphql-ws"
* Configuration file: none
* Routing: none

## Code Organization

* [ndn](ndn): NDN library in pure Go.
* [mk](mk): build helper scripts.
* [csrc](csrc): C source code.
* [js](js): TypeScript source code.
* [bpf](bpf): eBPF programs, such as forwarding strategies.
* [core](core): common shared code.
* [dpdk](dpdk): Go bindings for DPDK and SPDK.
* [ndni](ndni): NDN packet representation for internal use.
* [iface](iface): network interfaces.
* [container](container): data structures.
* [app](app): application level modules, such as the forwarder data plane.
* [cmd](cmd): executables.
* [sample](sample): control plane samples.
* [docs](docs): documentation.

There is a `README.md` file in most directories of this codebase that describes the corresponding module.
