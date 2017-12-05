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

	ot := dpdk.NewRingObjTable(2)
	defer ot.Close()

	nDequeued, nEntries := r.BurstDequeue(ot)
	assert.EqualValues(0, nDequeued)
	assert.EqualValues(0, nEntries)

	ot.Set(0, unsafe.Pointer(uintptr(9971)))
	ot.Set(1, unsafe.Pointer(uintptr(3087)))
	nEnqueued, freeSpace := r.BurstEnqueue(ot)
	assert.EqualValues(2, nEnqueued)
	assert.EqualValues(1, freeSpace)
	assert.EqualValues(2, r.Count())
	assert.EqualValues(1, r.GetFreeSpace())
	assert.False(r.IsEmpty())
	assert.False(r.IsFull())

	ot.Set(0, unsafe.Pointer(uintptr(2776)))
	ot.Set(1, unsafe.Pointer(uintptr(1876)))
	nEnqueued, freeSpace = r.BurstEnqueue(ot)
	assert.EqualValues(1, nEnqueued)
	assert.EqualValues(0, freeSpace)
	assert.EqualValues(3, r.Count())
	assert.EqualValues(0, r.GetFreeSpace())
	assert.False(r.IsEmpty())
	assert.True(r.IsFull())

	nDequeued, nEntries = r.BurstDequeue(ot)
	assert.EqualValues(2, nDequeued)
	assert.EqualValues(1, nEntries)
	assert.Equal(unsafe.Pointer(uintptr(9971)), ot.Get(0))
	assert.Equal(unsafe.Pointer(uintptr(3087)), ot.Get(1))
	assert.EqualValues(1, r.Count())
	assert.EqualValues(2, r.GetFreeSpace())

	nDequeued, nEntries = r.BurstDequeue(ot)
	assert.EqualValues(1, nDequeued)
	assert.EqualValues(0, nEntries)
	assert.Equal(unsafe.Pointer(uintptr(2776)), ot.Get(0))
	assert.EqualValues(0, r.Count())
	assert.EqualValues(3, r.GetFreeSpace())
}
