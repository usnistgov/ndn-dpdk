package ealthreadtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

func TestAllocator(t *testing.T) {
	defer ealthread.AllocClear()
	defer eal.UpdateLCoreSockets(map[int]int{0: 0, 1: 0, 2: 0, 3: 0, 4: 1, 5: 1, 6: 1}, 0)()

	assert, require := makeAR(t)
	lc1, lc2, lc3, lc4, lc5, lc6 := eal.LCoreFromID(1), eal.LCoreFromID(2), eal.LCoreFromID(3), eal.LCoreFromID(4), eal.LCoreFromID(5), eal.LCoreFromID(6)
	numa0, numa1 := eal.NumaSocketFromID(0), eal.NumaSocketFromID(1)

	m, e := ealthread.AllocConfig(ealthread.Config{
		"A": {
			LCores: []int{1, 6},
		},
		"B": {
			PerNuma: map[int]int{0: 1},
		},
		"C": {
			PerNuma: map[int]int{0: 1, 1: 2},
		},
	})
	require.NoError(e)
	assert.Len(m, 3)
	assert.ElementsMatch(eal.LCores{lc1, lc6}, m["A"])
	assert.Len(m["B"], 1)
	assert.Subset(eal.LCores{lc2, lc3}, m["B"])
	assert.Len(m["C"], 3)
	assert.Subset(m["C"], eal.LCores{lc4, lc5})
	assert.NotSubset(m["C"], m["B"])
	ealthread.AllocClear()

	list, e := ealthread.AllocRequest(
		ealthread.AllocReq{Role: "A", Socket: numa0},
		ealthread.AllocReq{Role: "A", Socket: numa1},
		ealthread.AllocReq{},
		ealthread.AllocReq{Role: "B"},
	)
	require.NoError(e)
	assert.Len(list, 4)
	assert.False(list[2].Valid())
	assert.Equal(numa0, list[3].NumaSocket())
}
