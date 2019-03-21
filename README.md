# High-Performance NDN Programs with DPDK

This repository contains high-performance [Named Data Networking (NDN)](https://named-data.net/) programs developed with [Data Plane Development Kit (DPDK)](https://www.dpdk.org/).

## Installation

Requirements:

* Ubuntu 16.04 or 18.04 on *amd64* architecture
* Go 1.11.5
* `clang-3.9 clang-format-3.9 curl doxygen git go-bindata libc6-dev-i386 libnuma-dev libssl-dev liburcu-dev pandoc socat sudo yamllint` packages
* DPDK 19.02 shared libraries installed to `/usr/local`; including OpenSSL PMD
* SPDK 19.01 shared libraries installed to `/usr/local`
* [ubpf](https://github.com/iovisor/ubpf/tree/10e0a45b11ea27696add38c33e24dbc631caffb6) library, installed to `/usr/local/include/ubpf.h` and `/usr/local/lib/libubpf.a`
* Node.js 11.x and `sudo npm install -g jayson`
* Note: see [Dockerfile](./Dockerfile) on how to install dependencies.

Build steps:

1. Clone repository into `$GOPATH/src/ndn-dpdk`.
2. Execute `make godeps` to compile C code and generate certain Go source files.
3. Execute `go get -d -t ./...` to download Go dependencies.
4. Execute `make cmds` to compile and install Go commands.
   They are in `$GOPATH/bin`, `./build/*.sh`, and `./cmd/nfdemu/build`.
5. Execute `npm install` to download NPM dependencies.
6. Execute `npm run build` to build TypeScript modules and commands.

Other build targets and commands:

* Execute `make` to build all Go packages.
* Execute `make test` or `./gotest.sh` to run unit tests.
* Execute `make docs` to build documentation.
  You may omit `doxygen pandoc` dependencies if this is not needed.
* Execute `./format-code.sh` to fix code style before committing.
  You may omit `clang-format-3.9 yamllint` dependencies if this is not needed.
* Prepend `RELEASE=1` to all `make` commands to select release mode that disables asserts and verbose logging.
* Note: you cannot use `go get` installation due to dependency between C code.

Docker packaging:

1. Build the image: `./build-docker.sh`
2. Launch a container in privileged mode: `docker run --rm -it --privileged -v /sys/bus/pci/devices:/sys/bus/pci/devices -v /sys/kernel/mm/hugepages:/sys/kernel/mm/hugepages -v /sys/devices/system/node:/sys/devices/system/node -v /dev:/dev --network host ndn-dpdk`
3. Setup environment inside the container: `mkdir /mnt/huge1G && mount -t hugetlbfs nodev /mnt/huge1G -o pagesize=1G && export PATH=$PATH:/usr/local/go/bin && export GOPATH=/root/go`
4. Only a subset of the programs would work in Docker container, unfortunately.

## Code Organization

* [core](core/): common shared code.
* [dpdk](dpdk/): DPDK bindings and extensions.
* [spdk](spdk/): SPDK bindings and extensions.
* [ndn](ndn/): NDN packet representations.
* [iface](iface/): network interfaces.
* [container](container/): data structures.
* [strategy](strategy/): forwarding strategy BPF programs.
* [app](app/): applications, including the forwarder dataplane.
* [mgmt](mgmt/): management interface.
* [appinit](appinit/): initialization procedures.
* [cmd](cmd/): executables.
