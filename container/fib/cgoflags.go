package fib

/*
#cgo CFLAGS: -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../../build -lndn-dpdk-tsht -lndn-dpdk-ndn -lndn-dpdk-dpdk -ldpdk -lurcu-qsbr -lurcu-cds
*/
import "C"
