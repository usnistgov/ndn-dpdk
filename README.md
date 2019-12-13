# NDN-DPDK: High-Speed Named Data Networking Forwarder

NDN-DPDK is a set of high-speed [Named Data Networking (NDN)](https://named-data.net/) programs developed with [Data Plane Development Kit (DPDK)](https://www.dpdk.org/). It includes a network forwarder and a traffic generator.

This software is developed at [Advanced Network Technologies Division](https://www.nist.gov/itl/antd) of [National Institute of Standards and Technology](https://www.nist.gov). It is in pre-release stage and will continue to be updated.

## Installation

Requirements:

* Ubuntu 16.04 or 18.04 on *amd64* architecture
* Go 1.13.5
* `clang-6.0 clang-format-6.0 curl doxygen gcc-7 git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev rake socat sudo yamllint` packages
  (add [ppa:ubuntu-toolchain-r/test](https://launchpad.net/~ubuntu-toolchain-r/+archive/ubuntu/test) on Ubuntu 16.04)
* [Intel Multi-Buffer Crypto for IPsec Library](https://github.com/intel/intel-ipsec-mb) v0.53
* DPDK 19.11 with [patch 63727](https://patches.dpdk.org/patch/63727/), with `CONFIG_RTE_BUILD_SHARED_LIB` `CONFIG_RTE_LIBRTE_BPF_ELF` `CONFIG_RTE_LIBRTE_PMD_OPENSSL` `CONFIG_RTE_LIBRTE_PMD_AESNI_MB` enabled, compiled with gcc-7, and installed to `/usr/local`
* SPDK 19.10 shared libraries, compiled with gcc-7, and installed to `/usr/local`
* [ubpf](https://github.com/iovisor/ubpf/tree/644ad3ded2f015878f502765081e166ce8112baf) library, compiled with gcc-7, and installed to `/usr/local/include/ubpf.h` and `/usr/local/lib/libubpf.a`
* Node.js 12.x and `sudo npm install -g jayson`
* Note: see [Dockerfile](./Dockerfile) on how to install dependencies.

Build steps:

1. Clone repository into `$GOPATH/src/ndn-dpdk`.
2. Execute `npm install` to download NPM dependencies.
3. Execute `make godeps` to compile C code and generate certain Go/TypeScript source files.
4. Execute `make goget` to download Go dependencies.
5. Execute `make cmds` to install Go commands to `$GOPATH/bin`.
6. Execute `make tsc` to build TypeScript modules and commands.

Other build targets and commands:

* Execute `sudo make install` to install commands to `/usr/local`, and `sudo make uninstall` to uninstall.
  You may prepend `DESTDIR=/opt` to choose a different location.
* Execute `make gopkg` to build all Go packages.
* Execute `make test` to run unit tests,  or `mk/gotest.sh PKG` to run tests for a package.
* Execute `make doxygen` to build C documentation.
  You may omit `doxygen` dependencies if this is not needed.
* Execute `make godoc` to start godoc server at port 6060.
* Execute `make lint` to fix code style before committing.
  You may omit `clang-format-6.0 yamllint` dependencies if this is not needed.
* Prepend `RELEASE=1` to any `make` command to select release mode that disables asserts and verbose logging.
* Prepend `CC=clang-6.0` to any `make` command to compile C code with `clang-6.0`.

Docker packaging:

1. Build the image: `mk/build-docker.sh`
2. Launch a container in privileged mode: `docker run --rm -it --privileged -v /sys/bus/pci/devices:/sys/bus/pci/devices -v /sys/kernel/mm/hugepages:/sys/kernel/mm/hugepages -v /sys/devices/system/node:/sys/devices/system/node -v /dev:/dev --network host ndn-dpdk`
3. Setup environment inside the container: `mkdir /mnt/huge1G && mount -t hugetlbfs nodev /mnt/huge1G -o pagesize=1G && export PATH=$PATH:/usr/local/go/bin && export GOPATH=/root/go`

## Code Organization

* [mk](mk/): build helper scripts.
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
