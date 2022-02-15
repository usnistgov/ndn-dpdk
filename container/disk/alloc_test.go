package disk_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/disk"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestAlloc(t *testing.T) {
	assert, require := makeAR(t)

	a := disk.NewAlloc(512, 1011, eal.NumaSocket{})
	defer a.Close()
	aMin, aMax := a.SlotRange()
	assert.EqualValues(512, aMin)
	assert.EqualValues(1011, aMax)

	slots := map[uint64]bool{}
	for i := 0; i < 500; i++ {
		slot, e := a.Alloc()
		if assert.NoError(e, i) {
			assert.LessOrEqual(uint64(512), slot, i)
			assert.GreaterOrEqual(uint64(1011), slot, i)
			assert.False(slots[slot], i)
			slots[slot] = true
		}
	}
	assert.Len(slots, 500)

	_, e := a.Alloc()
	assert.Error(e)

	a.Free(515)
	slot, e := a.Alloc()
	require.NoError(e)
	assert.Equal(uint64(515), slot)

	_, e = a.Alloc()
	assert.Error(e)
}

func TestSizeCalc(t *testing.T) {
	assert, _ := makeAR(t)

	calc := disk.SizeCalc{
		NThreads:   4,
		NPackets:   1000,
		PacketSize: 5000,
	}

	assert.Equal(10, calc.BlocksPerSlot())
	assert.Equal(int64(40010), calc.MinBlocks())

	a0 := calc.CreateAlloc(0, eal.NumaSocket{})
	defer a0.Close()
	min0, max0 := a0.SlotRange()
	assert.Equal(uint64(1), min0)
	assert.Equal(uint64(1000), max0)

	a3 := calc.CreateAlloc(3, eal.NumaSocket{})
	defer a3.Close()
	min3, max3 := a3.SlotRange()
	assert.Equal(uint64(3001), min3)
	assert.Equal(uint64(4000), max3)
}
