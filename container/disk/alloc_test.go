package disk_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestAlloc(t *testing.T) {
	assert, require := makeAR(t)

	a := disk.NewAlloc(512, 1023, eal.NumaSocket{})
	defer a.Close()

	slots := map[uint64]bool{}
	for i := 0; i < 512; i++ {
		slot, e := a.Alloc()
		require.NoError(e)
		assert.LessOrEqual(uint64(512), slot)
		assert.GreaterOrEqual(uint64(1023), slot)
		assert.False(slots[slot])
		slots[slot] = true
	}
	assert.Len(slots, 512)

	_, e := a.Alloc()
	assert.Error(e)

	a.Free(515)
	slot, e := a.Alloc()
	require.NoError(e)
	assert.Equal(uint64(515), slot)

	_, e = a.Alloc()
	assert.Error(e)
}
