package ndn

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../build-c -lndn-dpdk-dpdk -L/usr/local/lib -ldpdk
*/
import "C"
