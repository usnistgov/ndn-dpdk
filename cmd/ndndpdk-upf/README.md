# ndndpdk-upf

This command runs a PFCP server that turns NDN-DPDK forwarder into 5G UPF.
The PFCP-related implementation is in [package upf](../../app/upf).
The GTP-U face implementation in in [package ethface](../../iface/ethface).

This program shall be deployed alongside an NDN-DPDK forwarder on the same host.
To use this program:

1. Activate the NDN-DPDK service as a forwarder.
2. Create an Ethernet port on the Ethernet device intended for N3 interface.
3. If desired, create a fallback face on the Ethernet port, so that ARP and IP works on the N3 interface.
4. Start ndndpdk-upf to provide a PFCP server on the N4 interface, which would be ready for incoming messages from the SMF.

Command line flags of this program include:

* `--gqlserver`: GraphQL endpoint of NDN-DPDK service activated as forwarder.
* `--smf-n4`: SMF N4 IPv4 address.
  The UPF only accepts PFCP messages from this IP address.
* `--upf-n4`: UPF N4 IPv4 address.
  The UPF binds to this IP address while listening for PFCP messages.
  This IP address must be configured on a kernel network interface.
* `--upf-n3`: UPF N3 IPv4 address.
  NDN-DPDK forwarder uses this IP address as the local IP in outer IPv4 header of GTP-U packets.
* `--upf-mac`: UPF N3 MAC address, corresponding to `--upf-n3`.
  NDN-DPDK forwarder uses this MAC address as the local MAC in outer Ethernet header of GTP-U packets.
* `--upf-vlan`: UPF N3 VLAN ID.
  If set, NDN-DPDK forwarder inserts a VLAN header before the outer IPv4 header in GTP-U packets.
* `--n3`: N3 ip-mac tuples, repeatable.
  NDN-DPDK forwarder does not perform ARP lookups.
  Every remote IP address (usually belongs to gNBs) that could appear in PFCP session must be listed in these tuples.
  NDN-DPDK forwarder uses this MAC address as the remote MAC in outer Ethernet header of GTP-U packets.
* `--dn`: Data Network NDN forwarder IPv4 address.
  NDN-DPDK forwarder uses this IP address as the local IP in inner IPv4 header of GTP-U packets.
