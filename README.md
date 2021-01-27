# NDN-DPDK: High-Speed Named Data Networking Forwarder

NDN-DPDK is a set of high-speed [Named Data Networking (NDN)](https://named-data.net/) programs developed with the [Data Plane Development Kit (DPDK)](https://www.dpdk.org/).
Included are a network forwarder and a traffic generator.

![NDN-DPDK logo](docs/NDN-DPDK-logo.svg)

This software is developed at the [Advanced Network Technologies Division](https://www.nist.gov/itl/antd) of the [National Institute of Standards and Technology](https://www.nist.gov/).
It is in pre-release stage and will continue to be updated.

## Installation

### Requirements

* Ubuntu 18.04 or Debian 10 on *amd64* architecture
* Required APT packages: `build-essential clang-8 git libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev pkg-config python3-distutils`
* Optional APT packages: `clang-format-8 doxygen yamllint`
  (see "other build targets" for an explanation)
* [pip](https://pip.pypa.io/en/stable/installing/) and `sudo pip install -U meson ninja`
* [Intel Multi-Buffer Crypto for IPsec Library](https://github.com/intel/intel-ipsec-mb) v0.55 (optional)
* DPDK 20.11, configured with `meson -Ddebug=true -Doptimization=3 -Dtests=false --libdir=lib build`
* [SPDK](https://spdk.io/) 20.10, configured with `./configure --enable-debug --disable-tests --with-shared --with-dpdk=/usr/local --without-vhost --without-isal --without-fuse`
* [ubpf](https://github.com/iovisor/ubpf/tree/089f6279752adfb01386600d119913403ed326ee/vm) library, installed to `/usr/local`
* Go 1.x
* Node.js 14.x

You can install the dependencies with [ndndpdk-depends.sh](docs/ndndpdk-depends.sh).

NDN-DPDK requires hugepages to run.
See [huge-setup.sh](docs/huge-setup.sh) for an example on how to setup hugepages.

### Build steps

1. Clone the repository.
2. Execute `npm install` to download NPM dependencies.
3. Execute `make` to compile the project.
4. Execute `sudo make install` to install the programs to `/usr/local`, and `sudo make uninstall` to uninstall them.

### Other build targets

* `make godeps` builds C objects and generates certain Go source files.
* `make gopkg` builds all Go packages.
* `make test` runs all unit tests.
  You can also execute `mk/gotest.sh <PKG>` to run the tests for a given package.
* `make doxygen` builds C documentation (requires the `doxygen` dependency).
* To view Go documentation, execute `godoc &` and access the website on port 6060.
  You may need to install [godoc](https://pkg.go.dev/golang.org/x/tools/cmd/godoc) command: `sudo GO111MODULE=off GOBIN=$(go env GOROOT)/bin $(which go) get -u golang.org/x/tools/cmd/godoc`
* `make lint` fixes code style issues before committing (requires the `clang-format-8` and `yamllint` dependencies).

### Compile-time settings

You can change compile-time settings by setting these environment variables:

* `NDNDPDK_MK_RELEASE=1` selects release mode that disables assertions and verbose logging in C code.
* `NDNDPDK_MK_THREADSLEEP=1` inserts `nanosleep(1ns)` to each thread.
  This reduces performance significantly, but is occasionally useful when running on a machine with fewer CPU cores.
* C code other than strategy is compiled with `gcc` by default; you can override this by setting the `CC` environment variable.
* Strategy code is compiled with `clang-8` by default; you can override this by setting the `BPFCC` environment variable.

You must run `make clean` when switching compile-time settings.

### Docker packaging

1. Build the image: `docker build -t ndn-dpdk .`
2. Configure hugepages on the host machine: `echo 8 | sudo tee /sys/devices/system/node/node*/hugepages/hugepages-1048576kB/nr_hugepages && sudo mkdir -p /mnt/huge1G && sudo mount -t hugetlbfs nodev /mnt/huge1G -o pagesize=1G`
3. Launch a container in privileged mode: `docker run --rm -it --privileged --network host --mount type=bind,source=/mnt/huge1G,target=/mnt/huge1G ndn-dpdk`
4. Run NDN-DPDK service inside the container: `ndndpdk-svc`
5. Or run unit tests: `export PATH=$PATH:/usr/local/go/bin; cd /root/ndn-dpdk; make test`

Note that DPDK is compiled with `-march=native` flag, so that the Docker image only works on machines with the same CPU model.

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
