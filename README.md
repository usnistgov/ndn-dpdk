# NDN-DPDK: High-Speed Named Data Networking Forwarder

NDN-DPDK is a set of high-speed [Named Data Networking (NDN)](https://named-data.net/) programs developed with [Data Plane Development Kit (DPDK)](https://www.dpdk.org/). It includes a network forwarder and a traffic generator.

This software is developed at [Advanced Network Technologies Division](https://www.nist.gov/itl/antd) of [National Institute of Standards and Technology](https://www.nist.gov/). It is in pre-release stage and will continue to be updated.

## Installation

Requirements:

* Ubuntu 18.04 or Debian 10 on *amd64* architecture
* Required packages: `build-essential clang-8 curl git go-bindata libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev pkg-config python3-distutils rake sudo` packages
* Optional packages: `clang-format-8 doxygen yamllint`
  (see other build targets list for explanation)
* [pip](https://pip.pypa.io/en/stable/installing/) and `sudo pip install meson ninja`
* [Intel Multi-Buffer Crypto for IPsec Library](https://github.com/intel/intel-ipsec-mb) v0.53
* DPDK 20.05, configured with `meson -Dtests=false --libdir=lib build`
* SPDK 20.04.1, configured with `./configure --enable-debug --disable-tests --with-shared --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse`
* [ubpf](https://github.com/iovisor/ubpf/tree/089f6279752adfb01386600d119913403ed326ee/vm) library, installed to `/usr/local`
* Go 1.14.4 or newer
* Node.js 14.x
* [jsonrpc2client](https://github.com/powerman/rpc-codec/releases) 1.1.3 or newer, installed to `/usr/local/bin/jsonrpc2client`
* Note: see [Dockerfile](./Dockerfile) on how to install dependencies.

Build steps:

1. Clone the repository.
2. Execute `npm install` to download NPM dependencies.
3. Execute `make` to compile the project.
4. Execute `sudo make install` to install commands to `/usr/local`, and `sudo make uninstall` to uninstall.

Other build targets:

* Execute `make godeps` to build C objects and generate certain Go/TypeScript source files.
* Execute `make gopkg` to build all Go packages.
* Execute `make test` to run unit tests,  or `mk/gotest.sh PKG` to run tests for a package.
* Execute `make doxygen` to build C documentation (requires `doxygen` package).
* Execute `make godoc` to start godoc server at port 6060.
* Execute `make lint` to fix code style before committing (requires `clang-format-8 yamllint` package).
* To select release mode that disables asserts and verbose logging in C code, prepend `RELEASE=1`.
  You must run `make clean` when switching between debug and release modes.
* C code other than strategy is compiled with `gcc` by default; override by setting `CC` environment variable.
* Strategy code is compiled with `clang-8` by default; override by setting `BPFCC` environment variable.

Docker packaging:

1. Build the image: `mk/build-docker.sh`
2. Launch a container in privileged mode: `docker run --rm -it --privileged -v /sys/bus/pci/devices:/sys/bus/pci/devices -v /sys/kernel/mm/hugepages:/sys/kernel/mm/hugepages -v /sys/devices/system/node:/sys/devices/system/node -v /dev:/dev --network host ndn-dpdk`
3. Setup environment inside the container: `mkdir /mnt/huge1G && mount -t hugetlbfs nodev /mnt/huge1G -o pagesize=1G && export PATH=$PATH:/usr/local/go/bin`

## Code Organization

* [mk](mk/): build helper scripts.
* [csrc](csrc/): C source code.
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
