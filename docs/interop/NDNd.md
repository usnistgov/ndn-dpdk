# NDN-DPDK Interoperability with NDNd

NDN-DPDK is interoperable with recent version of [NDNd](https://github.com/named-data/ndnd), a Golang implementation of the NDN stack.
This page gives a few samples on how to establish communication between NDN-DPDK and NDNd forwarder (formerly YaNFD).

## Prepare NDNd Docker Image

When you follow through this guide, it is recommended to install NDNd as a Docker image.
This provides a clean environment for running NDNd forwarder, and avoids interference from other software you may have.
Once you have finished this guide, you can use the same procedures on other NDNd installations.

Dockerfile and related scripts are provided in [docs/interop/ndnd](ndnd) directory.
It compiles a recent version of NDNd, and generates configurations suitable for this guide.

To build the NDNd Docker image:

```bash
cd docs/interop/ndnd
docker build --pull -t localhost/ndnd .
```

The NDNd package has a forwarder but lacks a forwarder management program.
Its forwarder implements a subnet of NFD management protocol, and therefore can be used with NFD's management programs.
Thus, you also need to build the NFD container, as described on [NFD page](NFD.md).

NDN-DPDK should be installed as a systemd service, not a Docker container.

## UDP Unicast over a Link

```text
|--------|                                                  |--------|
|producer|                                                  |consumer|
| /net/A |---\    |---------|    UDP     |---------|    /---| /net/A |
|--------|    \---|NDN-DPDK |  unicast   |         |---/    |--------|
                  |forwarder|------------| NDNd fw |
|--------|    /---|  (A)    |            |   (B)   |---\    |--------|
|consumer|---/    |---------|            |---------|    \---|producer|
| /net/B |                                                  | /net/B |
|--------|                                                  |--------|
```

In this scenario, NDN-DPDK forwarder and NDNd forwarder on two separate machines communicate via UDP unicast:

* Node A runs NDN-DPDK forwarder, a producer for `/net/A` prefix, and a consumer for `/net/B` prefix.
* Node B runs NDNd forwarder, a producer for `/net/B` prefix, and a consumer for `/net/A` prefix.
* FIB entries are created on each forwarder so that the applications can communicate.
* This scenario assumes the two machines are directly connected, without intermediate IP routers.

This scenario uses the following variables.
You need to modify them to fit your hardware, and paste them on every terminal before entering commands.

```bash
# PCI address of the Ethernet adapter on node A
A_IF_PCI=04:00.0
# network interface name of the Ethernet adapter on node A
A_IFNAME=eth1
# hardware address of the Ethernet adapter on node A
A_HWADDR=02:00:00:00:00:01
# IP address of the Ethernet adapter on node A (either IPv4 or IPv6)
A_IP=fd54:450f:f5ac:807a::1/64
# name prefix of producer A
A_NAME=/net/A
# network interface name of the Ethernet adapter on node B
B_IFNAME=eth1
# hardware address of the Ethernet adapter on node B
B_HWADDR=02:00:00:00:00:02
# IP address of the Ethernet adapter on node B (same address family as A_IP)
B_IP=fd54:450f:f5ac:807a::2/64
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
A_IPPORT=$(echo $A_IP | awk -F/ '{ if ($1~":") { print "[" $1 "]:6363" } else { print $1 ":6363" } }')
B_IPPORT=$(echo $B_IP | awk -F/ '{ if ($1~":") { print "[" $1 "]:6363" } else { print $1 ":6363" } }')
A_FACEID=$(ndndpdk-ctrl create-udp-face --local $A_HWADDR --remote $B_HWADDR \
           --udp-local $A_IPPORT --udp-remote $B_IPPORT | tee /dev/stderr | jq -r .id)

# insert FIB entry
A_FIBID=$(ndndpdk-ctrl insert-fib --name $B_NAME --nh $A_FACEID | tee /dev/stderr | jq -r .id)

# start the producer
sudo mkdir -p /run/ndn
ndndpdk-godemo pingserver --name $A_NAME --payload 512
```

On node B, start NDNd forwarder and producer:

```bash
# bring up the Ethernet adapter and configure ARP/NDP entry
sudo ip link set $B_IFNAME up
sudo ip addr replace $B_IP dev $B_IFNAME
A_IPADDR=$(echo $A_IP | awk -F/ '{ print $1 }')
sudo ip neigh replace $A_IPADDR lladdr $A_HWADDR nud noarp dev $B_IFNAME

# stop NDNd forwarder if it's already running
docker rm -f yanfd

# start NDNd forwarder
docker volume create run-ndn
docker run -d --name yanfd \
  --network host --init \
  --mount type=volume,source=run-ndn,target=/run/nfd \
  localhost/ndnd

# make 'nfdc' alias
alias nfdc='docker run --rm --mount type=volume,source=run-ndn,target=/run/ndn localhost/nfd nfdc'

# create face
A_FACEURI=$(echo $A_IP | awk -F/ '{ if ($1~":") { print "udp6://[" $1 "]:6363" } else { print "udp4://" $1 ":6363" } }')
B_FACEID=$(nfdc face create remote $A_FACEURI persistency permanent \
           | tee /dev/stderr | awk -vRS=' ' -vFS='=' '$1=="id"{print $2}')

# insert route
nfdc route add prefix $A_NAME nexthop $B_FACEID

# start the producer
docker run -it --rm --network none \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  localhost/nfd \
  ndnpingserver --size 512 $B_NAME
```

On node A, start a consumer:

```bash
# run the consumer
ndndpdk-godemo pingclient --name ${B_NAME}/ping --interval 10ms
```

On node B, start a consumer:

```bash
# run the consumer
docker run -it --rm --network none \
  --mount type=volume,source=run-ndn,target=/var/run/nfd \
  localhost/ndnd \
  ping -i 10 $A_NAME
```
