package ealtest

import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestMalloc(t *testing.T) {
	assert, _ := makeAR(t)

	ptr1 := eal.Zmalloc[int]("unittest", 65536, eal.NumaSocket{})
	assert.NotNil(ptr1)
	defer eal.Free(ptr1)

	ptr2 := eal.ZmallocAligned[int]("unittest", 65536, 8, eal.NumaSocket{})
	assert.NotNil(ptr2)
	assert.Zero(uintptr(unsafe.Pointer(ptr2)) % 512)
	defer eal.Free(ptr2)
}
