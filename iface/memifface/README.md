# ndn-dpdk/iface/memifface

This package implements memif faces.
See [face creation](../../docs/face.md) "memif face" section for locator syntax.

The underlying implementation is in [package ethport](../ethport).

In the data plane:

* NDN-DPDK and application must operate its memif interface in opposite roles.
* Each packet is an NDN packet without Ethernet header.
* `MEMIF_DESC_FLAG_NEXT` may cause stability issue in the RX path.
  It is currently disabled in the TX path during face creation.
