package main

/*
#cgo CFLAGS: -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../../build-c -lndn-dpdk-iface -lndn-dpdk-ndn -lndn-dpdk-dpdk -lndn-dpdk-core -ldpdk
*/
import "C"
