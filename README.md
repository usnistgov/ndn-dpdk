# High-Performance NDN Programs with DPDK

This repository contains high-performance [Named Data Networking (NDN)](https://named-data.net/) programs developed with [Data Plane Development Kit (DPDK)](http://dpdk.org/).

## Installation

Requirements:

* Ubuntu 16.04 or 18.04 on *amd64* architecture
* Go 1.10.3
* `clang-3.9 clang-format-3.9 curl doxygen git go-bindata libc6-dev-i386 libnuma-dev libssl-dev liburcu-dev pandoc socat sudo yamllint` packages
* DPDK 18.05 shared libraries installed to `/usr/local`; including OpenSSL PMD
* [ubpf](https://github.com/iovisor/ubpf/tree/10e0a45b11ea27696add38c33e24dbc631caffb6) library, installed to `/usr/local/include/ubpf.h` and `/usr/local/lib/libubpf.a`
* NodeJS 8.x and `sudo npm install -g jayson tslint typescript`

Installation steps:

1. Clone repository to `$GOPATH/src/ndn-dpdk`.
2. Execute `go get -d -t ./...` inside the repository.
3. `make`, and have a look at other [Makefile](./Makefile) targets.
   Prepend `RELEASE=1` selects release mode that disables asserts and verbose logging.
   Note: `go get` installation is unavailable due to dependency between C code.

This program can also run in a Docker container in privileged mode.
The `Dockerfile` serves a useful reference on how to install dependencies.

1. Build the image: `./build-docker.sh`
2. Launch a container: `docker run --rm -it --privileged -v /sys/bus/pci/devices:/sys/bus/pci/devices -v /sys/kernel/mm/hugepages:/sys/kernel/mm/hugepages -v /sys/devices/system/node:/sys/devices/system/node -v /dev:/dev --network host ndn-dpdk`
3. Setup environment inside the container: `mkdir /mnt/huge1G && mount -t hugetlbfs nodev /mnt/huge1G -o pagesize=1G && export PATH=$PATH:/usr/local/go/bin && export GOPATH=/root/go`

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
