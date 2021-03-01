# NDN-DPDK Docker Packaging

This page describes how to build an NDN-DPDK Docker container with the provided [Dockerfile](../Dockerfile).

## Build the Image

The simplest command to build a Docker image is:

```bash
docker build -t ndn-dpdk .
```

Some DPDK drivers may require external dependencies.
For example, the mlx5 driver for Mellanox ConnectX-4/5/6 Ethernet adapters needs the `libibverbs-dev` package.
You can use `APT_PKGS` build argument to add external dependencies.

By default, the image is non-portable due to the use of `-march=native` compiler flag.
The [installation guide](INSTALL.md) "dependencies" section explains that you can pass `--arch=CPU-TYPE` argument to the ndndpdk-depends.sh script to change the target CPU architecture.
You can use `DEPENDS_ARGS` build argument to pass arguments to the script.

Example command to enable mlx5 driver and build for Skylake CPU:

```bash
docker build \
  --build-arg APT_PKGS="libibverbs-dev" --build-arg DEPENDS_ARGS="--arch=skylake" \
  -t ndn-dpdk .
```

## Start the Container

NDN-DPDK requires hugepages to run, and you may need to change PCI driver bindings to support certain hardware.
These must be configured on the host machine.
The [installation guide](INSTALL.md) "usage" section describes how to perform these tasks.

Example command to start a container for interactive use:

```bash
docker run -it --rm --name ndn-dpdk \
  --privileged --network host \
  --mount type=bind,source=/mnt/huge1G,target=/mnt/huge1G \
  ndn-dpdk
```

* `--privileged` enables privileged mode, which allows DPDK to interact with hugepages and PCI devices.
* `--network host` selects host networking, which allows DPDK to configure network stack.
* `--mount` mounts hugepages into the container.
  Depending on whether you are using 2MB or 1GB hugepages in the huge-setup.sh script, you may need to change the paths.

You can start the NDN-DPDK service within the container:

```bash
ndndpdk-svc
```

You can run unit tests:

```bash
cd /root/ndn-dpdk
make test
```

## Start NDN-DPDK Service as a Container

Example command to start a NDN-DPDK service container:

```bash
docker run -d --rm --name ndn-dpdk \
  --privileged --network host \
  --mount type=bind,source=/mnt/huge1G,target=/mnt/huge1G \
  ndn-dpdk ndndpdk-svc
```

You can then use the `ndndpdk-ctrl` command line tool as follows:

```bash
docker run -i --rm --network host ndn-dpdk ndndpdk-ctrl ARGUMENTS
```
