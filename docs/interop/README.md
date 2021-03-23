# NDN-DPDK Interoperability with Other NDN Software

NDN-DPDK implements the NDN protocols and can theoretically communicate with other NDN software that implement the same protocols.
In reality, existing implementations (including NDN-DPDK itself) have varying degrees of completeness.
If one implementation requires a certain protocol feature, but the other implementation does not have that feature, communication cannot succeed.
Moreover, differences in transports and management protocols can also create gaps in achieving interoperability.

Common issues include:

* NDN-DPDK requires upstream nodes and producer applications to support PIT token.
  Each outgoing Interest from NDN-DPDK carries a PIT token, which must be returned with the Data or Nack packet in reply to that Interest.
  However, many NDN implementations do not recognize PIT tokens.
* NDN-DPDK does not create a face upon an incoming connection attempt.
  Instead, face creation must be requested via the management API.
* NDN-DPDK uses a different management protocol from other NDN forwarders.

This page summarizes current knowledge of interoperability between NDN-DPDK and other NDN implementations.
If NDN-DPDK is interoperable with another NDN implementation, sample steps to achieve basic communication will be included.

## NDN Forwarding Daemon (NFD)

[NFD](https://named-data.net/doc/NFD/) v0.7.1 or later is interoperable with NDN-DPDK.
See [NDN-DPDK interoperability with NFD](NFD.md) for a few sample scenarios on how to establish communication.

## ndn-cxx and python-ndn

[ndn-cxx](https://named-data.net/doc/ndn-cxx/) and [python-ndn](https://python-ndn.readthedocs.io) are incompatible with NDN-DPDK.
They do not support PIT tokens, and do not support NDN-DPDK management protocol.

To use applications based on these libraries, you can run NFD alongside NDN-DPDK on the same machine.
In this case:

* Local applications can connect to NFD using their existing libraries.
* NFD handles packet forwarding from, to, and between local applications.
* NDN-DPDK handles packet forwarding from and to remote nodes.
* NFD has a minimal cache, while NDN-DPDK operates a larger cache.

See [NDN-DPDK interoperability with NFD](NFD.md) for a sample scenario.

## NDNts

[NDNts](https://yoursunny.com/p/NDNts/), when running in Node.js environment, is interoperable with NDN-DPDK.
It fully supports PIT tokens, and has partial integration with NDN-DPDK management API.

Applications must be import [`@ndn/dpdkmgmt` package](https://www.npmjs.com/package/@ndn/dpdkmgmt) in order to communicate with NDN-DPDK.
