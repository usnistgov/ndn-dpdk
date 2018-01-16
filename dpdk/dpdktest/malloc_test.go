package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestMalloc(t *testing.T) {
	assert, _ := makeAR(t)

	ptr1 := dpdk.Zmalloc("unittest", 65536, dpdk.NUMA_SOCKET_ANY)
	assert.NotNil(ptr1)
	defer dpdk.Free(ptr1)

	ptr2 := dpdk.ZmallocAligned("unittest", 65536, 8, dpdk.NUMA_SOCKET_ANY)
	assert.NotNil(ptr2)
	assert.Zero(uintptr(ptr2) % 512)
	defer dpdk.Free(ptr2)
}
