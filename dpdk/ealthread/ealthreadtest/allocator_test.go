package ealthreadtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

type mockLCoreProvider struct{}

func (mockLCoreProvider) Workers() []eal.LCore {
	return []eal.LCore{
		eal.LCoreFromID(1),
		eal.LCoreFromID(2),
		eal.LCoreFromID(3),
		eal.LCoreFromID(4),
		eal.LCoreFromID(5),
		eal.LCoreFromID(6),
		eal.LCoreFromID(7),
	}
}

func (mockLCoreProvider) IsBusy(lc eal.LCore) bool {
	if lc.ID() == 7 {
		return true
	}
	return false
}

func (mockLCoreProvider) NumaSocketOf(lc eal.LCore) eal.NumaSocket {
	if lc.ID() < 4 {
		return eal.NumaSocketFromID(0)
	}
	return eal.NumaSocketFromID(1)
}

func TestAllocator(t *testing.T) {
	assert, _ := makeAR(t)

	la := ealthread.NewAllocator(mockLCoreProvider{})
	la.Config = ealthread.AllocConfig{
		"A": {
			LCores:   []int{1, 6, 8},
			EachNuma: 2,
		},
		"B": {
			LCores: []int{4},
			OnNuma: map[int]int{0: 1},
		},
		"C": {
			LCores: []int{1},
			OnNuma: map[int]int{0: 3, 1: 4},
		},
	}
	numa0, numa1 := eal.NumaSocketFromID(0), eal.NumaSocketFromID(1)

	// 1=reserved-AC, 2=idle, 3=idle, 4=reserved-B, 5=idle, 6=reserved-A, 7=busy
	// pick from reserved-A on NUMA 0
	lc1 := la.Alloc("A", numa0)
	assert.Equal(1, lc1.ID())

	// 1=allocated-A, 2=idle, 3=idle, 4=reserved-B, 5=idle, 6=reserved-A, 7=busy
	// pick from reserved-A on NUMA 1
	lc6 := la.Alloc("A", numa1)
	assert.Equal(6, lc6.ID())

	// 1=allocated-A, 2=idle, 3=idle, 4=reserved-B, 5=idle, 6=allocated-A, 7=busy
	// pick from idle on NUMA 1
	lc5 := la.Alloc("A", numa1)
	assert.Equal(5, lc5.ID())

	// 1=allocated-A, 2=idle, 3=idle, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from idle on remote NUMA
	lc2 := la.Alloc("A", numa1)
	assert.Equal(2, lc2.ID())

	// 1=allocated-A, 2=allocated-A, 3=idle, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// fail because exceeding OnNuma limit
	assert.False(la.Alloc("A", numa1).Valid())

	// 1=allocated-A, 2=allocated-A, 3=idle, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from idle on NUMA 0
	lc3 := la.Alloc("B", numa0)
	assert.Equal(3, lc3.ID())

	// 1=allocated-A, 2=allocated-A, 3=allocated-B, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from reserved-B on remote NUMA
	lc4 := la.Alloc("B", numa0)
	assert.Equal(4, lc4.ID())

	// 1=allocated-A, 2=allocated-A, 3=allocated-B, 4=allocated-B, 5=allocated-A, 6=allocated-A, 7=busy
	// fail because no lcore available
	assert.False(la.Alloc("C", numa0).Valid())

	la.Free(lc2)

	// 1=allocated-A, 2=idle, 3=allocated-B, 4=allocated-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from reserved-A on NUMA 0
	lc2 = la.Alloc("C", numa0)
	assert.Equal(2, lc2.ID())
}
