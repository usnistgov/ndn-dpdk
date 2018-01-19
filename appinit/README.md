# ndn-dpdk/appinit

This package implements program initialization procedures.

* `exitcodes.go`: common error exit codes.
* `eal.go`: initialize DPDK EAL, launch lcores.
* `mempool.go`: create DPDK mempools on each NUMA socket.
* `face.go`: create NDN face by FaceUri.
