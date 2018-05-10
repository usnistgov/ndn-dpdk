# High-Performance NDN Programs with DPDK

This repository contains high-performance [Named Data Networking (NDN)](https://named-data.net/) programs developed with [Data Plane Development Kit (DPDK)](http://dpdk.org/).

## Installation

Requirements:

* Ubuntu 16.04 on `amd64` architecture
* `clang clang-format doxygen go-bindata libc6-dev-i386 liburcu-dev nodejs pandoc socat` packages
* DPDK 17.11, installed to `/usr/local`
* [ubpf](https://github.com/iovisor/ubpf/tree/10e0a45b11ea27696add38c33e24dbc631caffb6) library, installed to `/usr/local/include/ubpf.h` and `/usr/local/lib/libubpf.a`
* NodeJS 8.x and NPM `jayson` package

Installation steps:

1. Clone repository to `$GOPATH/src/ndn-dpdk`.
2. Execute `go get -t ./...` inside the repository.
3. `make`, and have a look at other [Makefile](./Makefile) targets.
   Prepend `RELEASE=1` selects release mode that disables asserts and verbose logging.
   Note: `go get` installation is unavailable due to dependency between C code.

## Code Organization

* [core](core/): common shared code.
* [dpdk](dpdk/): DPDK bindings and extensions.
* [ndn](ndn/): NDN packet representations.
* [iface](iface/): network interfaces.
* [container](container/): data structures.
* [strategy](strategy/): forwarding strategy BPF programs.
* [app](app/): applications.
* [mgmt](mgmt/): management interface.
* [appinit](appinit/): initialization procedures.
* [cmd](cmd/): executables.
