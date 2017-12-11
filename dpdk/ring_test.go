package dpdk

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"unsafe"
)

func TestRing(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	r, e := NewRing("TestRing", 4, GetCurrentLCore().GetNumaSocket(), true, true)
	require.NoError(e)
	defer r.Close()

	assert.EqualValues(0, r.Count())
	assert.EqualValues(3, r.GetFreeSpace())
	assert.True(r.IsEmpty())
	assert.False(r.IsFull())

	output := make([]unsafe.Pointer, 3)
	nDequeued, nEntries := r.BurstDequeue(output[:2])
	assert.EqualValues(0, nDequeued)
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

	nDequeued, nEntries = r.BurstDequeue(output[:1])
	assert.EqualValues(1, nDequeued)
	assert.EqualValues(2, nEntries)
	assert.Equal(unsafe.Pointer(uintptr(9971)), output[0])
	assert.EqualValues(2, r.Count())
	assert.EqualValues(1, r.GetFreeSpace())

	nDequeued, nEntries = r.BurstDequeue(output[:3])
	assert.EqualValues(2, nDequeued)
	assert.EqualValues(0, nEntries)
	assert.Equal(unsafe.Pointer(uintptr(3087)), output[0])
	assert.Equal(unsafe.Pointer(uintptr(2776)), output[1])
	assert.EqualValues(0, r.Count())
	assert.EqualValues(3, r.GetFreeSpace())
}
