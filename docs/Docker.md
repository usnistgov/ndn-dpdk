# NDN-DPDK Docker Container

This page describes how to run NDN-DPDK service in a Docker container.

## Build the Image

The simplest command to build a Docker image with the provided [Dockerfile](../Dockerfile) is:

```bash
docker build -t ndn-dpdk .
```

Some DPDK drivers may require external dependencies.
For example, the mlx5 driver for Mellanox ConnectX-4/5/6 Ethernet adapters needs the `libibverbs-dev` package.
You can use `APT_PKGS` build argument to add external dependencies.

By default, the image is non-portable due to the use of `-march=native` compiler flag.
The [installation guide](INSTALL.md) "dependencies" section explains that you can pass `--arch=CPU-TYPE` argument to the ndndpdk-depends.sh script to change the target CPU architecture.
You can use `DEPENDS_ARGS` build argument to pass arguments to the script.

By default, NDN-DPDK is built in debug mode.
The [installation guide](INSTALL.md) "compile-time settings" section explains that you can set `NDNDPDK_MK_RELEASE=1` environment variable to select release mode.
You can use `MAKE_ENV` build argument to pass environment variables to the Makefile.

Example command to enable mlx5 driver, target Skylake CPU, and select release mode:

```bash
docker build \
  --build-arg APT_PKGS="libibverbs-dev" \
  --build-arg DEPENDS_ARGS="--arch=skylake" \
  --build-arg MAKE_ENV="NDNDPDK_MK_RELEASE=1" \
  -t ndn-dpdk .
```

## Prepare the Host Machine

NDN-DPDK requires hugepages to run, and you may need to change PCI driver bindings to support certain hardware.
These must be configured on the host machine.
The [installation guide](INSTALL.md) "usage" section describes how to perform these tasks.

You can [download](https://core.dpdk.org/download/) DPDK setup scripts, or extract from the image:

```bash
CTID=$(docker container create ndn-dpdk)
for S in dpdk-devbind.py dpdk-hugepages.py; do
  docker cp $CTID:/usr/local/bin/$S - | sudo tar -x -C /usr/local/bin
done
docker rm $CTID
```

## Start the NDN-DPDK Service Container

Example command to start the NDN-DPDK service container:

```bash
sudo mkdir -p /run/ndndpdk-memif

docker run -d --name ndndpdk-svc \
  --cap-add IPC_LOCK --cap-add NET_ADMIN --cap-add NET_RAW --cap-add SYS_ADMIN --cap-add SYS_NICE \
  --device /dev/infiniband --device /dev/vfio \
  --mount type=bind,source=/dev/hugepages,target=/dev/hugepages \
  --mount type=bind,source=/run/ndndpdk-memif,target=/run/ndndpdk-memif \
  ndn-dpdk
```

### Explanation of Docker Flags

`--cap-add` adds capabilities required by DPDK.

`--device` allows access to PCI devices such as Ethernet adapters.
The required device list is hardware dependent; see [hardware known to work](hardware.md) for some examples.

`--mount target=/dev/hugepages` mounts hugepages into the container.

`--mount target=/run/ndndpdk-memif` shares a directory for memif control sockets.
You may use to use a Docker volume instead of a bind mount.
Applications using memif transport must set the memif *SocketName* to a socket in this directory.

## Control the NDN-DPDK Service Container

You can use standard Docker commands to control the container, such as:

```bash
# stop and delete the container
docker rm -f ndndpdk-svc

# restart the NDN-DPDK service
docker restart ndndpdk-svc

# view logs
docker logs -f ndndpdk-svc

# retrieve container IP address
docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ndndpdk-svc
```

You can access NDN-DPDK GraphQL endpoint on port 3030 of the container IP address.
It is not recommended to publish this port to the host machine, because the GraphQL service does not have authentication.

To use the `ndndpdk-ctrl` command line tool, create an alias:

```bash
alias ndndpdk-ctrl='docker run -i --rm \
  --add-host "ndndpdk-svc:$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ndndpdk-svc)" \
  ndn-dpdk ndndpdk-ctrl --gqlserver http://ndndpdk-svc:3030'
```

## Run Applications with Containerized NDN-DPDK Service

If the NDN-DPDK service container has been [activated as a forwarder](forwarder.md), you can run applications like this:

```bash
docker run --rm \
  --mount type=bind,source=/run/ndndpdk-memif,target=/run/ndndpdk-memif \
  --add-host "ndndpdk-svc:$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ndndpdk-svc)" \
  ndn-dpdk \
  ndndpdk-godemo --gqlserver http://ndndpdk-svc:3030 pingserver --name /example/P

docker run --rm \
  --mount type=bind,source=/run/ndndpdk-memif,target=/run/ndndpdk-memif \
  --add-host "ndndpdk-svc:$(docker inspect -f '{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}' ndndpdk-svc)" \
  ndn-dpdk \
  ndndpdk-godemo --gqlserver http://ndndpdk-svc:3030 pingclient --name /example/P
```

In the example commands:

* `--mount target=/run/ndndpdk-memif` shares a directory for memif control sockets.
* `--add-host` references the service container IP address.
* `--gqlserver` makes the demo application connect to the GraphQL endpoint in the service container instead of localhost.
