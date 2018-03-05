# ndn-dpdk/app/dump

This package implements a package dumper.
It retrieves L3 packets (C `Packet*` type) from a DPDK ring, prints packet names to a logger, and frees the mbufs.
