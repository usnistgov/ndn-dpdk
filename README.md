# High-Performance NDN Programs with DPDK

This repository contains high-performance [Named Data Networking (NDN)](https://named-data.net/) programs developed with [Data Plane Development Kit (DPDK)](https://www.dpdk.org/).

## Installation

Requirements:

* Ubuntu 16.04 or 18.04 on *amd64* architecture
* Go 1.12.7
* `clang-6.0 clang-format-6.0 curl doxygen git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev socat sudo yamllint` packages
* DPDK 19.08-rc2 with `CONFIG_RTE_BUILD_SHARED_LIB` `CONFIG_RTE_LIBRTE_BPF_ELF` `CONFIG_RTE_LIBRTE_PMD_OPENSSL` enabled, and installed to `/usr/local`
* SPDK 19.04.1 +[patch](https://github.com/spdk/spdk/commit/cf35beccf406200595dd63394a0d880850579916) shared libraries installed to `/usr/local`
* [ubpf](https://github.com/iovisor/ubpf/tree/644ad3ded2f015878f502765081e166ce8112baf) library, installed to `/usr/local/include/ubpf.h` and `/usr/local/lib/libubpf.a`
* Node.js 12.x and `sudo npm install -g jayson`
* Note: see [Dockerfile](./Dockerfile) on how to install dependencies.

Build steps:

1. Clone repository into `$GOPATH/src/ndn-dpdk`.
2. Execute `npm install` to download NPM dependencies.
3. Execute `make godeps` to compile C code and generate certain Go/TypeScript source files.
4. Execute `go get -d -t ./...` to download Go dependencies.
5. Execute `make cmds` to install Go commands to `$GOPATH/bin`.
6. Execute `make tsc` to build TypeScript modules and commands.

Other build targets and commands:

* Execute `make` to build all Go packages.
* Execute `make test` or `./gotest.sh` to run unit tests.
* Execute `make doxygen` to build C documentation.
  You may omit `doxygen` dependencies if this is not needed.
* Execute `make godoc` to start godoc server at port 6060.
* Execute `./format-code.sh` to fix code style before committing.
  You may omit `clang-format-6.0 yamllint` dependencies if this is not needed.
* Prepend `RELEASE=1` to any `make` command to select release mode that disables asserts and verbose logging.
* Prepend `CC=clang-6.0` to any `make` command to compile C code with `clang-6.0`.
  The programs are currently not working, but this is a good way to find potential code errors.
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
