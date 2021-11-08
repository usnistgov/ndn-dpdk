package fwdptest

import (
	"fmt"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
)

// fakeEthDevNumaSocket partially implements EthDev.
// It is acceptable to fwdp.DefaultAlloc function that only calls its NumaSocket() method.
type fakeEthDevNumaSocket struct {
	ethdev.EthDev
	socket eal.NumaSocket
}

func (dev fakeEthDevNumaSocket) NumaSocket() eal.NumaSocket {
	return dev.socket
}

func TestDefaultAllocFewLCores(t *testing.T) {
	assert, require := makeAR(t)
	defer eal.UpdateLCoreSockets(map[int]int{0: 0, 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 10: 1, 11: 1, 12: 1, 13: 1, 14: 1, 15: 0, 20: 0}, 20)()
	defer ealthread.AllocClear()

	numa0, numa1 := eal.NumaSocketFromID(0), eal.NumaSocketFromID(1)
	ethDevs := []ethdev.EthDev{
		fakeEthDevNumaSocket{nil, numa0},
		fakeEthDevNumaSocket{nil, numa0},
		fakeEthDevNumaSocket{nil, numa1},
	}

	alloc, e := fwdp.DefaultAlloc(ethDevs)
	require.NoError(e)
	fmt.Println(alloc)
	assert.Len(alloc[fwdp.RoleCrypto], 1)
	assert.Len(alloc[fwdp.RoleFwd], 5)
	assert.Len(alloc[fwdp.RoleInput], 3)
	assert.Len(alloc[fwdp.RoleOutput], 3)
	lcInput, lcOutput := alloc[fwdp.RoleInput].ByNumaSocket(), alloc[fwdp.RoleOutput].ByNumaSocket()
	assert.GreaterOrEqual(len(lcInput[numa0]), 1)
	assert.LessOrEqual(len(lcInput[numa1]), 1)
	assert.GreaterOrEqual(len(lcOutput[numa0]), 1)
	assert.LessOrEqual(len(lcOutput[numa1]), 1)
}

func TestDefaultAllocManyLCores(t *testing.T) {
	assert, require := makeAR(t)
	defer eal.UpdateLCoreSockets(
		map[int]int{0: 0, 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0, 7: 0,
			10: 1, 11: 1, 12: 1, 13: 1, 14: 1, 15: 1, 16: 1, 17: 1,
			20: 0}, 20)()
	defer ealthread.AllocClear()

	numa0, numa1 := eal.NumaSocketFromID(0), eal.NumaSocketFromID(1)
	ethDevs := []ethdev.EthDev{
		fakeEthDevNumaSocket{nil, numa0},
		fakeEthDevNumaSocket{nil, numa0},
		fakeEthDevNumaSocket{nil, numa1},
	}

	alloc, e := fwdp.DefaultAlloc(ethDevs)
	require.NoError(e)
	fmt.Println(alloc)
	assert.Len(alloc[fwdp.RoleCrypto], 1)
	assert.Len(alloc[fwdp.RoleFwd], 7)
	assert.Len(alloc[fwdp.RoleInput], 4)
	assert.Len(alloc[fwdp.RoleOutput], 4)
	lcInput, lcOutput := alloc[fwdp.RoleInput].ByNumaSocket(), alloc[fwdp.RoleOutput].ByNumaSocket()
	assert.GreaterOrEqual(len(lcInput[numa0]), 2)
	assert.GreaterOrEqual(len(lcInput[numa1]), 1)
	assert.GreaterOrEqual(len(lcOutput[numa0]), 2)
	assert.GreaterOrEqual(len(lcOutput[numa1]), 1)
}
