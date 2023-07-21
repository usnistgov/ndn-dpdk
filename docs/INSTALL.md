# NDN-DPDK Installation Guide

NDN-DPDK supports Ubuntu 22.04 and Debian 12 operating systems.
It only works on x64\_64 (amd64) architecture.

This page describes how to install and start NDN-DPDK on a supported operating system, which could be a physical server or a virtual machine with KVM acceleration.
You can also [build a Docker container](Docker.md), which would work on other operating systems.

## Dependencies

* Linux kernel 5.15 or newer
* Required APT packages: `clang-15 g++-12 git jq libc6-dev-i386 libelf-dev libpcap-dev libssl-dev liburcu-dev make ninja-build pkg-config`
* Optional APT packages: `clang-format-15 doxygen lcov yamllint`
* Go 1.20
* Node.js 20.x
* [Meson build system](https://mesonbuild.com/Getting-meson.html#installing-meson-with-pip)
* [ubpf](https://github.com/iovisor/ubpf) 7c6b8443
* [libbpf](https://github.com/libbpf/libbpf) 1.2.2 and [libxdp](https://github.com/xdp-project/xdp-tools) 1.2.10 (optional)
* [liburing](https://github.com/axboe/liburing) 2.4
* [Data Plane Development Kit (DPDK)](https://www.dpdk.org/) 23.03
* [Storage Performance Development Kit (SPDK)](https://spdk.io/) 23.05
* [godoc](https://pkg.go.dev/golang.org/x/tools/cmd/godoc) and [staticcheck](https://pkg.go.dev/honnef.co/go/tools/cmd/staticcheck) commands (optional)

You can run the [ndndpdk-depends.sh](ndndpdk-depends.sh) script to install these dependencies, or refer to the script for specific configuration options.
Certain hardware drivers may require installing extra dependencies before building DPDK or running the script; see [hardware known to work](hardware.md) for more information.

The script automatically downloads dependencies from the Internet.
If your network cannot reach certain download sites, you can specify a mirror site via `NDNDPDK_DL_*` environment variables.
See script source code for variable names and their default values.

## Build Steps

1. Clone the NDN-DPDK repository.
2. Run `corepack pnpm install` to download NPM dependencies.
3. Run `NDNDPDK_MK_RELEASE=1 make` to compile the project.
4. Run `sudo make install` to install the programs.

Installed files include:

* NDN-DPDK [commands](../cmd) in `/usr/local/bin`
* eBPF objects in `/usr/local/lib/bpf`
* bash completion scripts in `/usr/local/share/bash-completion/completions`
* configuration schemas and TypeScript definition in `/usr/local/share/ndn-dpdk`
* systemd template unit `ndndpdk-svc@.service`

## Usage

NDN-DPDK requires hugepages to run.
You may setup hugepages using the `dpdk-hugepages.py` script.
See [DPDK system requirements](https://doc.dpdk.org/guides/linux_gsg/sys_reqs.html#use-of-hugepages-in-the-linux-environment) for more information.

Depending on your hardware, you may need to change PCI driver bindings using the `dpdk-devbind.py` script.
See [DPDK Network Interface Controller Drivers](https://doc.dpdk.org/guides/nics/) and [hardware known to work](hardware.md) for more information.

You can run `sudo ndndpdk-ctrl systemd start` to start the NDN-DPDK service, use `ndndpdk-ctrl` command to activate it as a forwarder or some other role, and then control the service.
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
* show face list: `ndndpdk-ctrl --gqlserver http://127.0.0.1:3030 list-face`

## Other Build Targets

* `make godeps` builds C objects and generates certain Go source files.
* `make gopkg` builds all Go packages.
* `make test` runs all unit tests.
  You can also use `mk/gotest.sh <PKG>` to run the tests for a given package.
* `make doxygen` builds C documentation (requires the `doxygen` dependency).
* To view Go documentation, run `godoc &` and access the website on port 6060 (requires `godoc` dependency).
* `make lint` fixes code style issues before committing (requires `clang-format-15`, `staticcheck`, and `yamllint` dependencies).

## Compile-Time Settings

You can change compile-time settings by setting environment variables.
Unless indicated otherwise, you must run `make clean` when switching compile-time settings.

`NDNDPDK_MK_RELEASE=1` environment variable selects release mode that disables assertions and verbose logging in C code.

`NDNDPDK_MK_THREADSLEEP=1` environment variable causes a polling thread to sleep for a short duration if it processed zero packets in a loop iteration.
This reduces CPU utilization when running on a machine with fewer CPU cores, but may impair performance.

`NDNDPDK_MK_COVERAGE=1` environment variable enables C code coverage collection.
After running unit tests, you can generate coverage report with `make coverage` (requires `lcov` dependency).

`CC` environment variable specifies a compiler for C code, excluding eBPF.
The default is `gcc`.

`BPFCC` environment variable specifies a compiler for eBPF programs.
The default is `clang-11`.

C code (including DPDK and SPDK, excluding eBPF) is compiled with `-march=native` flag by default.
It selects the CPU instruction sets available on the local machine, and makes the compiled binaries incompatible with any other CPU model.
Pass `--arch=`*CPU-type* argument to the `ndndpdk-depends.sh` to change the target CPU architecture.
See [GCC - x86 options](https://gcc.gnu.org/onlinedocs/gcc/x86-Options.html) for available options.
To switch this setting, you need to rerun the dependency installation script and rebuild NDN-DPDK.

`GOAMD64` environment variable selects the x86-64 architecture level for Go code.
See [Go Minimum Requirements - amd64](https://github.com/golang/go/wiki/MinimumRequirements#amd64) for available options.
The default is `GOAMD64=v2`.
It may be overridden to a higher level such as `GOAMD64=v3`.
