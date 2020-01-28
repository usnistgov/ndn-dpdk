# NDN-DPDK: High-Speed Named Data Networking Forwarder

NDN-DPDK is a set of high-speed [Named Data Networking (NDN)](https://named-data.net/) programs developed with [Data Plane Development Kit (DPDK)](https://www.dpdk.org/). It includes a network forwarder and a traffic generator.

This software is developed at [Advanced Network Technologies Division](https://www.nist.gov/itl/antd) of [National Institute of Standards and Technology](https://www.nist.gov). It is in pre-release stage and will continue to be updated.

## Installation

Requirements:

* Ubuntu 16.04 or 18.04 on *amd64* architecture
* Required packages: `build-essential clang-6.0 curl gcc-7 git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev ninja-build pkg-config python3.8 python3-distutils rake socat sudo` packages
  (Ubuntu 16.04: add [ppa:ubuntu-toolchain-r/test](https://launchpad.net/~ubuntu-toolchain-r/+archive/ubuntu/test) and [ppa:deadsnakes/ppa](https://launchpad.net/~deadsnakes/+archive/ubuntu/ppa), change `python3-distutils` to `python3.8-distutils`)
* Optional packages: `clang-format-6.0 doxygen yamllint`
  (see other build targets list for explanation)
* [pip](https://pip.pypa.io/en/stable/installing/) and `sudo pip install meson`
* [Intel Multi-Buffer Crypto for IPsec Library](https://github.com/intel/intel-ipsec-mb) v0.53
* DPDK 19.11 with [patch 65156](https://patches.dpdk.org/patch/65156/), [patch 65158](https://patches.dpdk.org/patch/65158/), [patch 65270](https://patches.dpdk.org/patch/65270/), configured with `CC=gcc-7 meson -Dtests=false --libdir=lib build`
* SPDK 19.10.1, configured with `CC=gcc-7 ./configure --enable-debug --disable-tests --with-shared --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse`
* [ubpf](https://github.com/iovisor/ubpf/tree/644ad3ded2f015878f502765081e166ce8112baf) library, compiled with gcc-7, and installed to `/usr/local/include/ubpf.h` and `/usr/local/lib/libubpf.a`
* Go 1.13.7 or newer
* Node.js 12.x and `sudo npm install -g jayson`
* Note: see [Dockerfile](./Dockerfile) on how to install dependencies.

Build steps:

1. Clone repository into `$GOPATH/src/ndn-dpdk`.
2. Execute `npm install` to download NPM dependencies.
3. Execute `make godeps` to compile C code and generate certain Go/TypeScript source files.
4. Execute `make goget` to download Go dependencies.
5. Execute `make cmds` to install Go commands to `$GOPATH/bin`.
6. Execute `make tsc` to build TypeScript modules and commands.

Other build targets:

* Execute `sudo make install` to install commands to `/usr/local`, and `sudo make uninstall` to uninstall.
  You may prepend `DESTDIR=/opt` to choose a different location.
* Execute `make gopkg` to build all Go packages.
* Execute `make test` to run unit tests,  or `mk/gotest.sh PKG` to run tests for a package.
* Execute `make doxygen` to build C documentation (requires `doxygen` package).
* Execute `make godoc` to start godoc server at port 6060.
* Execute `make lint` to fix code style before committing (requires `clang-format-6.0 yamllint` package).
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
