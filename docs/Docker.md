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
The [installation guide](INSTALL.md) "dependencies" section explains that you can pass `--arch=CPU-TYPE` command line argument to the ndndpdk-depends.sh script to change the target CPU architecture.
Moreover, the script allows configuring download mirrors via environment variables.
You can use `DEPENDS_ENV` and `DEPENDS_ARGS` build arguments to pass environment variables and command line arguments to the script.

By default, NDN-DPDK is built in debug mode.
The [installation guide](INSTALL.md) "compile-time settings" section explains that you can set `NDNDPDK_MK_RELEASE=1` environment variable to select release mode.
You can use `MAKE_ENV` build argument to pass environment variables to the Makefile.

Example command to enable mlx5 driver, use alternate GOPROXY, target Skylake CPU, and select release mode:

```bash
docker build \
  --build-arg APT_PKGS="libibverbs-dev" \
  --build-arg DEPENDS_ENV="GOPROXY=https://goproxy.io,direct" \
  --build-arg DEPENDS_ARGS="--arch=skylake" \
  --build-arg MAKE_ENV="NDNDPDK_MK_RELEASE=1" \
  -t ndn-dpdk .
```

## Prepare the Host Machine

NDN-DPDK requires hugepages to run, and you may need to change PCI driver bindings to support certain hardware.
These must be configured on the host machine.
The [installation guide](INSTALL.md) "usage" section describes how to perform these tasks.

You can extract DPDK setup scripts and NDN-DPDK management schemas from the image:

```bash
sudo mkdir -p /usr/local/bin /usr/local/share
CTID=$(docker container create ndn-dpdk)
docker cp $CTID:/usr/local/bin/dpdk-devbind.py - | sudo tar -x -C /usr/local/bin
docker cp $CTID:/usr/local/bin/dpdk-hugepages.py - | sudo tar -x -C /usr/local/bin
docker cp $CTID:/usr/local/share/ndn-dpdk - | sudo tar -x -C /usr/local/share
docker container rm $CTID
```

## Start the NDN-DPDK Service Container

Example command to start the NDN-DPDK service container:

```bash
docker volume create run-ndn
docker run -d --name ndndpdk-svc \
  --restart on-failure \
  --cap-add IPC_LOCK --cap-add NET_ADMIN --cap-add SYS_ADMIN --cap-add SYS_NICE \
  --mount type=bind,source=/dev/hugepages,target=/dev/hugepages \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  ndn-dpdk

# retrieve container IP address for NDN-DPDK GraphQL endpoint
GQLSERVER=$(docker inspect -f 'http://{{range.NetworkSettings.Networks}}{{.IPAddress}}{{end}}:3030' ndndpdk-svc)
```

You can view logs from the NDN-DPDK service container with `docker logs -f ndndpdk-svc` command.

### Explanation of Docker Flags

`--restart on-failure` automatically restarts the service upon failure or `ndndpdk-ctrl shutdown --restart`.

`--cap-add` adds capabilities required by DPDK.

`--mount target=/dev/hugepages` mounts hugepages into the container.

`--mount target=/run/ndn` shares a volume for memif control sockets.
Applications using memif transport must set the memif *SocketName* to a socket in this directory.

Certain hardware may require additional `--device` and `--mount` flags.
See [hardware known to work](hardware.md) for some examples.

## Control the NDN-DPDK Service Container

You can use standard Docker commands to control the container, such as:

```bash
# stop and delete the container
docker rm -f ndndpdk-svc

# restart the NDN-DPDK service
docker restart ndndpdk-svc

# view logs
docker logs -f ndndpdk-svc
```

You can access NDN-DPDK GraphQL endpoint on port 3030 of the container IP address.
It is not recommended to publish this port to the host machine, because the GraphQL service does not have authentication.

To use the `ndndpdk-ctrl` command line tool, create an alias:

```bash
alias ndndpdk-ctrl='docker run -i --rm ndn-dpdk ndndpdk-ctrl --gqlserver $GQLSERVER'
```

## Run Applications with Containerized NDN-DPDK Service

If the NDN-DPDK service container has been [activated as a forwarder](forwarder.md), you can run applications like this:

```bash
docker run --rm \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  ndn-dpdk \
  ndndpdk-godemo --gqlserver $GQLSERVER pingserver --name /example/P

docker run --rm \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  ndn-dpdk \
  ndndpdk-godemo --gqlserver $GQLSERVER pingclient --name /example/P
```

In the example commands:

* `--mount target=/run/ndn` shares a volume for memif control sockets.
* `--gqlserver` makes the demo application connect to the GraphQL endpoint in the service container instead of localhost.
