package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

type mockLCoreProvider struct{}

func (mockLCoreProvider) ListSlaves() []dpdk.LCore {
	return []dpdk.LCore{
		dpdk.LCoreFromID(1),
		dpdk.LCoreFromID(2),
		dpdk.LCoreFromID(3),
		dpdk.LCoreFromID(4),
		dpdk.LCoreFromID(5),
		dpdk.LCoreFromID(6),
		dpdk.LCoreFromID(7),
	}
}

func (mockLCoreProvider) IsBusy(lc dpdk.LCore) bool {
	if lc.ID() == 7 {
		return true
	}
	return false
}

func (mockLCoreProvider) GetNumaSocket(lc dpdk.LCore) dpdk.NumaSocket {
	if lc.ID() < 4 {
		return dpdk.NumaSocketFromID(0)
	}
	return dpdk.NumaSocketFromID(1)
}

func TestLCoreAllocator(t *testing.T) {
	assert, _ := makeAR(t)

	var la dpdk.LCoreAllocator
	la.Provider = mockLCoreProvider{}
	la.Config = make(dpdk.LCoreAllocConfig)
	la.Config["A"] = dpdk.LCoreAllocRoleConfig{
		LCores:  []int{1, 6, 8},
		PerNuma: map[int]int{-1: 2},
	}
	la.Config["B"] = dpdk.LCoreAllocRoleConfig{
		LCores:  []int{4},
		PerNuma: map[int]int{0: 1},
	}
	la.Config["C"] = dpdk.LCoreAllocRoleConfig{
		LCores:  []int{1},
		PerNuma: map[int]int{0: 3, 1: 4},
	}

	numa0 := dpdk.NumaSocketFromID(0)
	numa1 := dpdk.NumaSocketFromID(1)

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
	// fail because exceeding PerNuma limit
	assert.False(la.Alloc("A", numa1).IsValid())

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
	assert.False(la.Alloc("C", numa0).IsValid())

	la.Free(lc2)

	// 1=allocated-A, 2=idle, 3=allocated-B, 4=allocated-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from reserved-A on NUMA 0
	lc2 = la.Alloc("C", numa0)
	assert.Equal(2, lc2.ID())
}
