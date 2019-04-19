package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

type mockLCoreProvider struct{}

func (mockLCoreProvider) ListSlaves() []dpdk.LCore {
	return []dpdk.LCore{1, 2, 3, 4, 5, 6, 7}
}

func (mockLCoreProvider) GetState(lc dpdk.LCore) dpdk.LCoreState {
	if lc == 7 {
		return dpdk.LCORE_STATE_RUNNING
	}
	return dpdk.LCORE_STATE_WAIT
}

func (mockLCoreProvider) GetNumaSocket(lc dpdk.LCore) dpdk.NumaSocket {
	if lc < 4 {
		return 0
	}
	return 1
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

	// 1=reserved-AC, 2=idle, 3=idle, 4=reserved-B, 5=idle, 6=reserved-A, 7=busy
	// pick from reserved-A on NUMA 0
	assert.Equal(dpdk.LCore(1), la.Alloc("A", 0))

	// 1=allocated-A, 2=idle, 3=idle, 4=reserved-B, 5=idle, 6=reserved-A, 7=busy
	// pick from reserved-A on NUMA 1
	assert.Equal(dpdk.LCore(6), la.Alloc("A", 1))

	// 1=allocated-A, 2=idle, 3=idle, 4=reserved-B, 5=idle, 6=allocated-A, 7=busy
	// pick from idle on NUMA 1
	assert.Equal(dpdk.LCore(5), la.Alloc("A", 1))

	// 1=allocated-A, 2=idle, 3=idle, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from idle on remote NUMA
	assert.Equal(dpdk.LCore(2), la.Alloc("A", 1))

	// 1=allocated-A, 2=allocated-A, 3=idle, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// fail because exceeding PerNuma limit
	assert.Equal(dpdk.LCORE_INVALID, la.Alloc("A", 1))

	// 1=allocated-A, 2=allocated-A, 3=idle, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from idle on NUMA 0
	assert.Equal(dpdk.LCore(3), la.Alloc("B", 0))

	// 1=allocated-A, 2=allocated-A, 3=allocated-B, 4=reserved-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from reserved-B on remote NUMA
	assert.Equal(dpdk.LCore(4), la.Alloc("B", 0))

	// 1=allocated-A, 2=allocated-A, 3=allocated-B, 4=allocated-B, 5=allocated-A, 6=allocated-A, 7=busy
	// fail because no lcore available
	assert.Equal(dpdk.LCORE_INVALID, la.Alloc("C", 0))

	la.Free(2)

	// 1=allocated-A, 2=idle, 3=allocated-B, 4=allocated-B, 5=allocated-A, 6=allocated-A, 7=busy
	// pick from reserved-A on NUMA 0
	assert.Equal(dpdk.LCore(2), la.Alloc("C", 0))
}
