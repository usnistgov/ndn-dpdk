# ndn-dpdk/iface/ifacetestenv

This package provides a test fixture for [ethface](../ethface/) and [socketface](../socketface/).
The calling test case must initialize the EAL, and create two faces that are connected together.
The fixture sends L3 packets on one face, and expects to receive them on the other face.
