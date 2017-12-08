package main

/*
#cgo CFLAGS: -m64 -pthread -O3 -march=native -I/usr/local/include/dpdk

#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
)

var t *integ.Testing
var mp dpdk.PktmbufPool

func main() {
	t = new(integ.Testing)
	defer t.Close()

	_, e := dpdk.NewEal([]string{"testprog", "-n1"})
	require.NoError(t, e)

	mp, e = dpdk.NewPktmbufPool("MP", 63, 0, 0, 1000, dpdk.NUMA_SOCKET_ANY)
	require.NoError(t, e)
	require.NotNil(t, mp)
	defer mp.Close()

	testMempool()
	testSegment()
	testPacket()
}
