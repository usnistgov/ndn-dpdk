package ndnping

/*
#cgo CFLAGS: -m64 -pthread -O3 -g -march=native -I/usr/local/include/dpdk
#cgo LDFLAGS: -L../../build -lndn-dpdk-iface -lndn-dpdk-nameset -lndn-dpdk-ndn -lndn-dpdk-dpdk -lndn-dpdk-core -ldpdk
*/
import "C"
