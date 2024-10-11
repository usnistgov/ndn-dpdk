# ndn-dpdk/app/upf

This package provides a 5G User Plane Function (UPF) that converts PFCP sessions to GTP-U faces.
It works as follows:

1. Listen for PFCP messages from a 5G Session Management Function (SMF).
2. Gather PFCP session related messages, convert them into NDN-DPDK face creation/deletion commands with appropriate locators of GTP-U faces.
3. These commands can then be sent to a running NDN-DPDK forwarder, to achieve NDN forwarding within a 5G network.

Currently, this package supports these PFCP message types:

* heartbeat request/response
* association setup request/response
* session establishment request/response
* session modification request/response
* session deletion request/response

For each message, it only recognizes the most basic fields required for constructing GTP-U headers, but does not support all cases.
It is tested to be compatible with several open-source SMF implementations, including free5GC and OAI-CN5G.
