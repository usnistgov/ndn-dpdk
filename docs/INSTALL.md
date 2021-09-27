# NDN-DPDK Installation Guide

NDN-DPDK supports Ubuntu 18.04, Ubuntu 20.04, and Debian 11 operating systems.
It only works on x64\_64 (amd64) architecture.

This page describes how to install and start NDN-DPDK on a supported operating system, which could be a physical server or a virtual machine with KVM acceleration.
You can also [build a Docker container](Docker.md), which would work on other operating systems.

## Dependencies

* Linux kernel 5.4 or newer (install `linux-image-generic-hwe-18.04` on Ubuntu 18.04)
* Required APT packages: `build-essential clang-11 git jq libc6-dev-i386 libelf-dev libpcap-dev libssl-dev liburcu-dev ninja-build pkg-config` (enable [llvm-toolchain-bionic-11](https://apt.llvm.org/) repository on Ubuntu 18.04)
* Optional APT packages: `clang-format-11 doxygen yamllint`
* Go 1.17
* Node.js 16.x
* [Meson build system](https://mesonbuild.com/Getting-meson.html#installing-meson-with-pip)
* [ubpf](https://github.com/iovisor/ubpf)
* [libbpf](https://github.com/libbpf/libbpf) 0.5.0 (optional)
* [liburing](https://github.com/axboe/liburing) 2.1
* [Data Plane Development Kit (DPDK)](https://www.dpdk.org/) 21.08
* [Storage Performance Development Kit (SPDK)](https://spdk.io/) 21.07
* [godoc](https://pkg.go.dev/golang.org/x/tools/cmd/godoc) and [staticcheck](https://pkg.go.dev/honnef.co/go/tools/cmd/staticcheck) commands (optional)
* [gq](https://www.npmjs.com/package/graphqurl) command (optional, only used in sample commands)

You can run the [ndndpdk-depends.sh](ndndpdk-depends.sh) script to install these dependencies, or refer to the script for specific configuration options.
Certain hardware drivers may require installing extra dependencies before building DPDK or running the script; see [hardware known to work](hardware.md) for more information.

By default, DPDK and SPDK are compiled with `-march=native` flag to maximize performance.
Binaries built this way are non-portable and can only work on machines with the same CPU model.
You can pass `--arch=CPU-TYPE` argument to the script to change the target CPU architecture.
*CPU-TYPE* should be set to the oldest CPU architecture you want to support, see [GCC - x86 options](https://gcc.gnu.org/onlinedocs/gcc/x86-Options.html) for available options.

The script automatically downloads dependencies from the Internet.
If your network cannot reach certain download sites, you can specify a mirror site via `NDNDPDK_DL_*` environment variables.
See script source code for variable names and their default values.

## Build Steps

1. Clone the NDN-DPDK repository.
2. Run `npm install` to download NPM dependencies.
3. Run `NDNDPDK_MK_RELEASE=1 make` to compile the project.
4. Run `sudo make install` to install the programs, and `sudo make uninstall` to uninstall them.

Installed files include:

* NDN-DPDK [commands](../cmd) in `/usr/local/bin`
* eBPF objects in `/usr/local/lib/bpf`
* systemd template unit `ndndpdk-svc@.service`
* configuration schemas and TypeScript definition in `/usr/local/share/ndn-dpdk`

## Usage

NDN-DPDK requires hugepages to run.
You may setup hugepages using the `dpdk-hugepages.py` script.
See [DPDK system requirements](https://doc.dpdk.org/guides/linux_gsg/sys_reqs.html#use-of-hugepages-in-the-linux-environment) for more information.

Depending on your hardware, you may need to change PCI driver bindings using the `dpdk-devbind.py` script.
See [DPDK Network Interface Controller Drivers](https://doc.dpdk.org/guides/nics/) and [hardware known to work](hardware.md) for more information.

You can then run `sudo ndndpdk-ctrl systemd start` to start the NDN-DPDK service, use `ndndpdk-ctrl` command to activate it as a forwarder or some other role, and then control the service.
See [forwarder activation and usage](forwarder.md), [traffic generator activation and usage](trafficgen.md), [file server activation and usage](fileserver.md) for basic usage in each role.
You can view logs from the NDN-DPDK service with `ndndpdk-ctrl systemd logs -f` command, which is especially useful in case of errors during activation and face creation.

As an alternative of using `ndndpdk-ctrl`, you can run queries and mutations on the GraphQL endpoint.
See [ndndpdk-ctrl](../cmd/ndndpdk-ctrl) for more information.

### Running Multiple Instances

NDN-DPDK installs a systemd template unit `ndndpdk-svc@.service`.
The template can be instantiated multiple times, with `host:port` of the GraphQL listener as the instance parameter.
For example, `ndndpdk-svc@127.0.0.1:3030.service` refers to an NDN-DPDK service instance listening on `http://127.0.0.1:3030`.

To successfully run multiple instances of NDN-DPDK service, it's necessary to ensure:

* Each GraphQL listener has a distinct `host:port`.
* Each instance has a distinct DPDK "file prefix", which is specified in `.eal.filePrefix` option of activation parameters.
* If using [CPU isolation](tuning.md), each instance has a distinct set of CPU cores.
* GraphQL commands are sent to the correct instance.

The `ndndpdk-ctrl` command accepts `--gqlserver` flag to specify the target instance.
This flag must appear between `ndndpdk-ctrl` and the subcommand name.
For example:

* start the service: `sudo ndndpdk-ctrl --gqlserver http://127.0.0.1:3030 systemd start`
* view service logs: `ndndpdk-ctrl --gqlserver http://127.0.0.1:3030 systemd logs -f`
* show face list: `ndndpdk-ctrl --gqlserver http://127.0.0.1:3030 list-faces`

## Other Build Targets

* `make godeps` builds C objects and generates certain Go source files.
* `make gopkg` builds all Go packages.
* `make test` runs all unit tests.
  You can also use `mk/gotest.sh <PKG>` to run the tests for a given package.
* `make doxygen` builds C documentation (requires the `doxygen` dependency).
* To view Go documentation, run `godoc &` and access the website on port 6060 (requires `godoc` dependency).
* `make lint` fixes code style issues before committing (requires `clang-format-11`, `staticcheck`, and `yamllint` dependencies).

## Compile-Time Settings

You can change compile-time settings by setting these environment variables:

* `NDNDPDK_MK_RELEASE=1` selects release mode that disables assertions and verbose logging in C code.
* `NDNDPDK_MK_THREADSLEEP=1` causes a polling thread to sleep for a short duration if it processed zero packets in a loop iteration.
  This reduces CPU utilization when running on a machine with fewer CPU cores, but may negatively impact performance.
* C code (except eBPF) is compiled with `gcc` by default; you can override this by setting the `CC` environment variable.
* eBPF programs are compiled with `clang-11` by default; you can override this by setting the `BPFCC` environment variable.

You must run `make clean` when switching compile-time settings.
