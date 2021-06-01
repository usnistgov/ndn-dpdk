# NDN-DPDK Interoperability with YaNFD

NDN-DPDK is interoperable with recent version of [YaNFD](https://github.com/named-data/YaNFD), Yet another NDN
Forwarder.
This page gives a few samples on how to establish communication between NDN-DPDK and YaNFD.

## Prepare YaNFD Docker Image

When you follow through this guide, it is recommended to install YaNFD as a Docker image.
This provides a clean environment for running YaNFD, and avoids interference from other software you may have.
Once you have finished this guide, you can use the same procedures on other YaNFD installations.

Dockerfile and related scripts are provided in [docs/interop/yanfd](yanfd) directory.
It compiles latest version of YaNFD, and generates configurations suitable for this guide.

To build the YaNFD Docker image:

```bash
cd docs/interop/yanfd
docker build -t yanfd .
```

The YaNFD package only contains a forwarder, and does not contain a management program.
The YaNFD forwarder implements a subnet of NFD management protocol, and therefore can be used with NFD's management programs.
Thus, you also need to build the NFD container, as described on [NFD page](NFD.md).

## UDP Unicast over a Link

```text
|--------|                                                  |--------|
|producer|                                                  |consumer|
| /net/A |---\    |---------|    UDP     |---------|    /---| /net/A |
|--------|    \---|NDN-DPDK |  unicast   |         |---/    |--------|
                  |forwarder|------------|  YaNFD  |
|--------|    /---|  (A)    |            |   (B)   |---\    |--------|
|consumer|---/    |---------|            |---------|    \---|producer|
| /net/B |                                                  | /net/B |
|--------|                                                  |--------|
```

In this scenario, NDN-DPDK forwarder and YaNFD on two separate machines communicate via UDP unicast:

* Node A runs NDN-DPDK forwarder, a producer for `/net/A` prefix, and a consumer for `/net/B` prefix.
* Node B runs YaNFD, a producer for `/net/B` prefix, and a consumer for `/net/A` prefix.
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
# bring up the network interface and assign IP address
sudo ip link set $A_IFNAME up
sudo ip addr replace $A_IP dev $A_IFNAME

# (re)start NDN-DPDK service
sudo systemctl restart ndndpdk-svc

# activate NDN-DPDK forwarder with PCI Ethernet adapter driver
# (this only works with bifurcated driver such as mlx5, because NDN-DPDK relies on the kernel to respond to ARP/NDP queries)
jq -n --arg if_pci $A_IF_PCI '
{
  eal: {
    coresPerNuma: { "0": 4, "1": 4 },
    pciDevices: [$if_pci]
  }
}' | ndndpdk-ctrl activate-forwarder

# or, activate NDN-DPDK forwarder with AF_XDP Ethernet adapter driver
jq -n '
{
  eal: {
    coresPerNuma: { "0": 4, "1": 4 }
  }
}' | ndndpdk-ctrl activate-forwarder

# create face
A_IPPORT=$(echo $A_IP | awk -F/ '{ if ($1~":") { print "[" $1 "]:6363" } else { print $1 ":6363" } }')
B_IPPORT=$(echo $B_IP | awk -F/ '{ if ($1~":") { print "[" $1 "]:6363" } else { print $1 ":6363" } }')
A_FACEID=$(ndndpdk-ctrl create-udp-face --local $A_HWADDR --remote $B_HWADDR \
           --udp-local $A_IPPORT --udp-remote $B_IPPORT | tee /dev/stderr | jq -r .id)

# insert FIB entry
A_FIBID=$(ndndpdk-ctrl insert-fib --name $B_NAME --nh $A_FACEID | tee /dev/stderr | jq -r .id)

# start the producer
sudo build/bin/ndndpdk-godemo pingserver --name $A_NAME --payload 512
```

On node B, start YaNFD and producer:

```bash
# bring up the network interface and assign IP address
sudo ip link set $B_IFNAME up
sudo ip addr replace $B_IP dev $B_IFNAME

# stop YaNFD if it's already running
docker rm -f yanfd

# start YaNFD
docker volume create run-ndn
docker run -d --rm --name yanfd \
  --network host \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  yanfd

# make 'nfdc' alias
alias nfdc='docker run --rm --mount type=volume,source=run-ndn,target=/run/ndn \
            -e NDN_CLIENT_TRANSPORT=unix:///run/ndn/yanfd.sock nfd nfdc'

# create face
A_FACEURI=$(echo $A_IP | awk -F/ '{ if ($1~":") { print "udp6://[" $1 "]:6363" } else { print "udp4://" $1 ":6363" } }')
B_FACEID=$(nfdc face create remote $A_FACEURI persistency permanent \
           | tee /dev/stderr | awk -vRS=' ' -vFS='=' '$1=="id"{print $2}')

# insert route
nfdc route add prefix $A_NAME nexthop $B_FACEID

# start the producer
docker run -it --rm --network none \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  -e NDN_CLIENT_TRANSPORT=unix:///run/ndn/yanfd.sock \
  nfd \
  ndnpingserver --size 512 $B_NAME
```

On node A, start a consumer:

```bash
# run the consumer
sudo build/bin/ndndpdk-godemo pingclient --name ${B_NAME}/ping --interval 10ms
```

On node B, start a consumer:

```bash
# run the consumer
docker run -it --rm --network none \
  --mount type=volume,source=run-ndn,target=/run/ndn \
  -e NDN_CLIENT_TRANSPORT=unix:///run/ndn/yanfd.sock \
  nfd \
  ndnping -i 10 $A_NAME
```
