# NDN-DPDK Interoperability with NDN Forwarding Daemon (NFD)

NDN-DPDK is interoperable with [NFD](https://named-data.net/doc/NFD/) v0.7.1 or later.
This page gives a few samples on how to establish communication between NDN-DPDK and NFD.

## Prepare NFD Docker Image

When you follow through this guide, it is recommended to install NFD as a Docker image.
This provides a clean environment for running NFD, and avoids interference from other software you may have.
Once you have finished this guide, you can use the same procedures on other NFD installations.

Dockerfile and related scripts are provided in [docs/interop/nfd](nfd) directory.
It installs the latest NFD version from the [NFD nightly APT repository](https://nfd-nightly.ndn.today/), and generates configurations suitable for this guide.

To build the NFD Docker image:

```bash
cd docs/interop/nfd
docker build -t nfd .
```

NDN-DPDK should be installed as a systemd service, not a Docker container.

## Ethernet Unicast over a Link

```text
|--------|                                                  |--------|
|producer|                                                  |consumer|
| /net/A |---\    |---------|  Ethernet  |---------|    /---| /net/A |
|--------|    \---|NDN-DPDK |  unicast   |         |---/    |--------|
                  |forwarder|------------|   NFD   |
|--------|    /---|  (A)    |            |   (B)   |---\    |--------|
|consumer|---/    |---------|            |---------|    \---|producer|
| /net/B |                                                  | /net/B |
|--------|                                                  |--------|
```

In this scenario, NDN-DPDK forwarder and NFD on two separate machines communicate via Ethernet unicast:

* Node A runs NDN-DPDK forwarder, a producer for `/net/A` prefix, and a consumer for `/net/B` prefix.
* Node B runs NFD, a producer for `/net/B` prefix, and a consumer for `/net/A` prefix.
* FIB entries are created on each forwarder so that the applications can communicate.

This scenario uses the following variables.
You need to modify them to fit your hardware, and paste them on every terminal before entering commands.

```bash
# PCI address of the Ethernet adapter on node A
A_IF_PCI=04:00.0
# hardware address of the Ethernet adapter on node A
A_HWADDR=02:00:00:00:00:01
# name prefix of producer A
A_NAME=/net/A
# network interface name of the Ethernet adapter on node B
B_IFNAME=eth1
# hardware address of the Ethernet adapter on node B
B_HWADDR=02:00:00:00:00:02
# name prefix of producer B
B_NAME=/net/B
```

On node A, start NDN-DPDK forwarder and producer:

```bash
# (re)start NDN-DPDK service
sudo ndndpdk-ctrl systemd restart

# activate NDN-DPDK forwarder
jq -n '
{
  eal: {
    coresPerNuma: { "0": 4, "1": 4 }
  }
}' | ndndpdk-ctrl activate-forwarder

# create Ethernet port with PCI driver
ndndpdk-ctrl create-eth-port --pci $A_IF_PCI

# create face
A_FACEID=$(ndndpdk-ctrl create-ether-face --local $A_HWADDR --remote $B_HWADDR | tee /dev/stderr | jq -r .id)

# insert FIB entry
A_FIBID=$(ndndpdk-ctrl insert-fib --name $B_NAME --nh $A_FACEID | tee /dev/stderr | jq -r .id)

# start the producer
sudo ndndpdk-godemo pingserver --name $A_NAME --payload 512
```

On node B, start NFD and producer:

```bash
# stop NFD if it's already running
docker rm -f nfd

# start NFD
docker volume create run-ndn
docker run -d --rm --name nfd \
  --cap-add=NET_ADMIN --network none --init \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  -e 'NFD_ENABLE_ETHER=1' \
  nfd

# activate the Ethernet adapter in NFD
B_CTPID=$(docker inspect -f '{{.State.Pid}}' nfd)
sudo ip link set $B_IFNAME netns $B_CTPID
sudo nsenter -t $B_CTPID -n ip link set $B_IFNAME up
docker exec nfd pkill -SIGHUP nfd

# make 'nfdc' alias
alias nfdc='docker exec nfd nfdc'

# create face
B_FACEID=$(nfdc face create local dev://${B_IFNAME} remote ether://[${A_HWADDR}] persistency permanent \
           | tee /dev/stderr | awk -vRS=' ' -vFS='=' '$1=="id"{print $2}')

# insert route
nfdc route add prefix $A_NAME nexthop $B_FACEID

# start the producer
docker run -it --rm --network none \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  nfd \
  ndnpingserver --size 512 $B_NAME
```

On node A, start a consumer:

```bash
# run the consumer
sudo ndndpdk-godemo pingclient --name ${B_NAME}/ping --interval 10ms
```

On node B, start a consumer:

```bash
# run the consumer
docker run -it --rm --network none \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  nfd \
  ndnping -i 10 $A_NAME
```

## Local Communication over Unix Socket

```text
|--------|        |---------|    Unix    |---------|        |--------|
|consumer|---\    |NDN-DPDK |   socket   |         |    /---|producer|
| /app/B |    \---|forwarder|------------|   NFD   |---/    | /app/B |
|--------|        |  (A)    |            |   (B)   |        |--------|
                  |---------|            |---------|
```

In this scenario, NDN-DPDK forwarder and NFD run on the same machine:

* NDN-DPDK and NFD communicate over a Unix socket.
* Producer for prefix `/app/B` connects to NFD.
* Consumer for prefix `/app/B` connects to NDN-DPDK.

This scenario uses the following variables.
You need to paste them on every terminal before entering commands.

```bash
# name prefix of producer in NFD
B_NAME=/app/B
```

Start NFD and producer:

```bash
# stop NFD if it's already running
docker rm -f nfd

# start NFD
docker volume create run-ndn
docker run -d --rm --name nfd \
  --network none --init \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  -e 'NFD_CS_CAP=1024' \
  nfd

# start the producer
docker run -it --rm \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  nfd \
  ndnpingserver --size 512 $B_NAME
```

Start and activate NDN-DPDK forwarder: see [forwarder activation](../forwarder.md).

Connect NDN-DPDK to NFD and run consumer:

```bash
# expose run-ndn volume on host machine
sudo mkdir -p /run/ndn
sudo mount --bind $(docker volume inspect -f '{{.Mountpoint}}' run-ndn) /run/ndn

# create face
A_FACEID=$(jq -n '{
  scheme: "unix",
  remote: "/run/ndn/nfd.sock"
}' | ndndpdk-ctrl create-face | tee /dev/stderr | jq -r .id)

# insert FIB entry
A_FIBID=$(ndndpdk-ctrl insert-fib --name $B_NAME --nh $A_FACEID | tee /dev/stderr | jq -r .id)

# run the consumer
sudo ndndpdk-godemo pingclient --name ${B_NAME}/ping --interval 10ms
# press CTRL+C to stop the consumer

# erase FIB entry and destroy face
ndndpdk-ctrl erase-fib --id $A_FIBID
ndndpdk-ctrl destroy-face --id $A_FACEID
```

The procedure above runs a producer on NFD side and a consumer on NDN-DPDK side.
Notice that it is necessary to create a FIB entry from NDN-DPDK to NFD, because each forwarder has a separate control plane.

It is also possible to run in the reverse direction, with a consumer on NFD side and a producer on NDN-DPDK side.
In that case, you need to initiate face creation from NDN-DPDK side, let NFD accept the incoming connection and automatically create a face, and then insert the route on NFD side.
One caveat is that you have to identify which Unix socket face on NFD side refers to the connection to NDN-DPDK, which could require some log parsing or guesswork.
