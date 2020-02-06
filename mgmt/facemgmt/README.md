# ndn-dpdk/mgmt/facemgmt

This package implements [face](../../iface/) management.

## Face

**Face.List** lists existing faces.

**Face.Get** retrieves information and counters of a specific face.

**Face.Create** creates a face.

**Face.Destroy** destroys a face.

## EthFace

**EthFace.ListPorts** lists Ethernet ports, including active and inactive ports.

**EthFace.ListPortFaces** lists Ethernet faces on a port.

**EthFace.ReadPortStats** reads DPDK statistics information from an Ethernet port.
