package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestCptrArray(t *testing.T) {
	assert, _ := makeAR(t)

	assert.Panics(func() { dpdk.ParseCptrArray(1) })
	assert.Panics(func() { dpdk.ParseCptrArray("x") })
	assert.Panics(func() { dpdk.ParseCptrArray([]string{"x", "y"}) })

	_, count := dpdk.ParseCptrArray([]cIntPtr{})
	assert.Equal(0, count)

	ptr, count := dpdk.ParseCptrArray([]cIntPtr{getCIntPtr(0), getCIntPtr(1)})
	assert.Equal(2, count)
	assert.True(checkCIntPtrArray(ptr))
}
