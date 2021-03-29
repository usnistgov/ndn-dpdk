# NDN-DPDK Installation Guide

NDN-DPDK supports Ubuntu 18.04, Ubuntu 20.04, and Debian 10 operating systems.
It only works on x64\_64 (amd64) architecture.

This page describes how to install and start NDN-DPDK on a supported operating system.
You can also [build a Docker container](Docker.md), which would work on other operating systems.

## Dependencies

* Required APT packages: `build-essential clang-8 git libc6-dev-i386 libelf-dev libnuma-dev libssl-dev liburcu-dev pkg-config python3-distutils`
* Optional APT packages: `clang-format-8 doxygen jq yamllint`
* Go 1.16
* Node.js 14.x
* Python 3, [pip](https://pip.pypa.io/en/stable/installing/), and PyPI packages: `meson ninja`
* [Intel Multi-Buffer Crypto for IPsec Library](https://github.com/intel/intel-ipsec-mb) v0.55 (optional)
* [Data Plane Development Kit (DPDK)](https://www.dpdk.org/) 20.11
* [Storage Performance Development Kit (SPDK)](https://spdk.io/) 20.10
* [ubpf](https://github.com/iovisor/ubpf) library, installed to `/usr/local`

You can execute the [ndndpdk-depends.sh](ndndpdk-depends.sh) script to install these dependencies, or refer to this script for the specific configuration options.
Certain hardware drivers may require installing extra dependencies before building DPDK or running the script; see [hardware known to work](hardware.md) for more information.

By default, DPDK and SPDK are compiled with `-march=native` flag to maximize performance.
Binaries built this way are non-portable and can only work on machines with the same CPU model.
You can pass `--arch=CPU-TYPE` argument to the script to change the target CPU architecture.
*CPU-TYPE* should be set to the oldest CPU architecture you want to support, see [gcc - x86 options](https://gcc.gnu.org/onlinedocs/gcc/x86-Options.html) for available options.

## Build Steps

1. Clone the NDN-DPDK repository.
2. Execute `npm install` to download NPM dependencies.
3. Execute `NDNDPDK_MK_RELEASE=1 make` to compile the project.
4. Execute `sudo make install` to install the programs, and `sudo make uninstall` to uninstall them.

Installed files include:

* NDN-DPDK [commands](../cmd) in `/usr/local/bin` and `/usr/local/sbin`
* eBPF objects in `/usr/local/lib/bpf`
* systemd service `ndndpdk-svc.service`
* configuration schemas and TypeScript definition in `/usr/local/share/ndn-dpdk`

## Usage

NDN-DPDK requires hugepages to run.
You may setup hugepages using the `dpdk-hugepages.py` script.
See [DPDK system requirements](https://doc.dpdk.org/guides/linux_gsg/sys_reqs.html#use-of-hugepages-in-the-linux-environment) for more information.

Depending on your hardware, you may need to change PCI driver bindings using the `dpdk-devbind.py` script.
See [DPDK Network Interface Controller Drivers](https://doc.dpdk.org/guides/nics/) for more information.

You can then execute `sudo systemctl start ndndpdk-svc` to start the NDN-DPDK service, use `ndndpdk-ctrl` command to activate it as a forwarder or a traffic generator, and then control the service.
See [forwarder activation and usage](forwarder.md) for sample commands to perform common operations on the forwarder.

NDN-DPDK service provides a GraphQL endpoint at `http://127.0.0.1:3030/` (changeable via `--gqlserver` argument).
As an alternative of using `ndndpdk-ctrl`, you can execute queries and mutations on the GraphQL endpoint.
The GraphQL service schema may be discovered via introspection.

Some GraphQL examples in NDN-DPDK documentation requires [graphqurl](https://github.com/hasura/graphqurl) command-line GraphQL client.
Execute `sudo npm install -g graphqurl` to install this command.

## Other Build Targets

* `make godeps` builds C objects and generates certain Go source files.
* `make gopkg` builds all Go packages.
* `make test` runs all unit tests.
  You can also execute `mk/gotest.sh <PKG>` to run the tests for a given package.
* `make doxygen` builds C documentation (requires the `doxygen` dependency).
* To view Go documentation, execute `godoc &` and access the website on port 6060.
  You may need to install [godoc](https://pkg.go.dev/golang.org/x/tools/cmd/godoc) command: `( cd /tmp && go get -u golang.org/x/tools/cmd/godoc )`.
* `make lint` fixes code style issues before committing (requires the `clang-format-8` and `yamllint` dependencies).

## Compile-Time Settings

You can change compile-time settings by setting these environment variables:

* `NDNDPDK_MK_RELEASE=1` selects release mode that disables assertions and verbose logging in C code.
* `NDNDPDK_MK_THREADSLEEP=1` inserts `nanosleep(1ns)` to each thread.
  This reduces performance significantly, but is occasionally useful when running on a machine with fewer CPU cores.
* C code (except eBPF) is compiled with `gcc` by default; you can override this by setting the `CC` environment variable.
* eBPF programs are compiled with `clang-8` by default; you can override this by setting the `BPFCC` environment variable.

You must run `make clean` when switching compile-time settings.
