package main

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"ndn-traffic-dpdk/dpdk"
	"ndn-traffic-dpdk/integ"
	"unsafe"
)

func main() {
	t := new(integ.Testing)
	defer t.Close()
	assert := assert.New(t)
	require := require.New(t)

	_, e := dpdk.NewEal([]string{"testprog", "-n1"})
	require.NoError(e)

	r, e := dpdk.NewRing("TestRing", 4, dpdk.GetCurrentLCore().GetNumaSocket(), true, true)
	require.NoError(e)
	defer r.Close()

	assert.EqualValues(0, r.Count())
	assert.EqualValues(3, r.GetFreeSpace())
	assert.True(r.IsEmpty())
	assert.False(r.IsFull())

	dequeued, nEntries := r.BurstDequeue(2)
	assert.EqualValues(0, len(dequeued))
	assert.EqualValues(0, nEntries)

	input := []unsafe.Pointer{unsafe.Pointer(uintptr(9971)), unsafe.Pointer(uintptr(3087))}
	nEnqueued, freeSpace := r.BurstEnqueue(input)
	assert.EqualValues(2, nEnqueued)
	assert.EqualValues(1, freeSpace)
	assert.EqualValues(2, r.Count())
	assert.EqualValues(1, r.GetFreeSpace())
	assert.False(r.IsEmpty())
	assert.False(r.IsFull())

	input = []unsafe.Pointer{unsafe.Pointer(uintptr(2776)), unsafe.Pointer(uintptr(1876))}
	nEnqueued, freeSpace = r.BurstEnqueue(input)
	assert.EqualValues(1, nEnqueued)
	assert.EqualValues(0, freeSpace)
	assert.EqualValues(3, r.Count())
	assert.EqualValues(0, r.GetFreeSpace())
	assert.False(r.IsEmpty())
	assert.True(r.IsFull())

	dequeued, nEntries = r.BurstDequeue(1)
	assert.EqualValues(1, len(dequeued))
	assert.EqualValues(2, nEntries)
	assert.Equal([]unsafe.Pointer{unsafe.Pointer(uintptr(9971))}, dequeued)
	assert.EqualValues(2, r.Count())
	assert.EqualValues(1, r.GetFreeSpace())

	dequeued, nEntries = r.BurstDequeue(3)
	assert.EqualValues(2, len(dequeued))
	assert.EqualValues(0, nEntries)
	assert.Equal([]unsafe.Pointer{unsafe.Pointer(uintptr(3087)), unsafe.Pointer(uintptr(2776))},
		dequeued)
	assert.EqualValues(0, r.Count())
	assert.EqualValues(3, r.GetFreeSpace())
}
