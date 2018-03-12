package dpdktest

import (
	"testing"
	"unsafe"

	"ndn-dpdk/dpdk"
)

func TestRing(t *testing.T) {
	assert, require := makeAR(t)

	r, e := dpdk.NewRing("TestRing", 4, dpdk.GetCurrentLCore().GetNumaSocket(), true, true)
	require.NoError(e)
	defer r.Close()

	assert.Equal(0, r.Count())
	assert.Equal(3, r.GetFreeSpace())
	assert.True(r.IsEmpty())
	assert.False(r.IsFull())
	assert.Equal(r.GetFreeSpace(), r.GetCapacity())

	output := make([]unsafe.Pointer, 3)
	nDequeued, nEntries := r.BurstDequeue(output[:2])
	assert.Equal(0, nDequeued)
	assert.Equal(0, nEntries)

	input := []unsafe.Pointer{unsafe.Pointer(uintptr(9971)), unsafe.Pointer(uintptr(3087))}
	nEnqueued, freeSpace := r.BurstEnqueue(input)
	assert.Equal(2, nEnqueued)
	assert.Equal(1, freeSpace)
	assert.Equal(2, r.Count())
	assert.Equal(1, r.GetFreeSpace())
	assert.False(r.IsEmpty())
	assert.False(r.IsFull())

	input = []unsafe.Pointer{unsafe.Pointer(uintptr(2776)), unsafe.Pointer(uintptr(1876))}
	nEnqueued, freeSpace = r.BurstEnqueue(input)
	assert.Equal(1, nEnqueued)
	assert.Equal(0, freeSpace)
	assert.Equal(3, r.Count())
	assert.Equal(0, r.GetFreeSpace())
	assert.False(r.IsEmpty())
	assert.True(r.IsFull())

	nDequeued, nEntries = r.BurstDequeue(output[:1])
	assert.Equal(1, nDequeued)
	assert.Equal(2, nEntries)
	assert.Equal(unsafe.Pointer(uintptr(9971)), output[0])
	assert.Equal(2, r.Count())
	assert.Equal(1, r.GetFreeSpace())

	nDequeued, nEntries = r.BurstDequeue(output[:3])
	assert.Equal(2, nDequeued)
	assert.Equal(0, nEntries)
	assert.Equal(unsafe.Pointer(uintptr(3087)), output[0])
	assert.Equal(unsafe.Pointer(uintptr(2776)), output[1])
	assert.Equal(0, r.Count())
	assert.Equal(3, r.GetFreeSpace())
}
