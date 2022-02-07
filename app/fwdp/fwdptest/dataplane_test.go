package fwdptest

import (
	"fmt"
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

func TestDefaultAllocInsufficientLCores(t *testing.T) {
	assert, _ := makeAR(t)
	defer eal.UpdateLCoreSockets(map[int]int{1: 0, 2: 0, 3: 0, 20: 0}, 20)()
	defer ealthread.AllocClear()

	_, e := fwdp.DefaultAlloc()
	assert.Error(e)
}

func TestDefaultAllocFewLCores(t *testing.T) {
	assert, require := makeAR(t)
	defer eal.UpdateLCoreSockets(map[int]int{
		1: 0, 2: 0, 3: 0,
		11: 1, 12: 1, 13: 1,
		20: 0}, 20)()
	defer ealthread.AllocClear()

	alloc, e := fwdp.DefaultAlloc()
	require.NoError(e)
	fmt.Println(alloc)
	assert.Len(alloc[fwdp.RoleInput], 2)
	assert.Len(alloc[fwdp.RoleOutput], 1)
	assert.Len(alloc[fwdp.RoleCrypto], 1)
	assert.Len(alloc[fwdp.RoleDisk], 0)
	assert.Len(alloc[fwdp.RoleFwd], 2)
}

func TestDefaultAllocManyLCores(t *testing.T) {
	assert, require := makeAR(t)
	defer eal.UpdateLCoreSockets(
		map[int]int{0: 0, 1: 0, 2: 0, 3: 0, 4: 0, 5: 0, 6: 0, 7: 0,
			10: 1, 11: 1, 12: 1, 13: 1, 14: 1, 15: 1, 16: 1, 17: 1,
			20: 0}, 20)()
	defer ealthread.AllocClear()
	numa0, numa1 := eal.NumaSocketFromID(0), eal.NumaSocketFromID(1)

	alloc, e := fwdp.DefaultAlloc()
	require.NoError(e)
	fmt.Println(alloc)
	assert.Len(alloc[fwdp.RoleInput], 4)
	assert.Len(alloc[fwdp.RoleOutput], 4)
	assert.Len(alloc[fwdp.RoleCrypto], 1)
	assert.Len(alloc[fwdp.RoleDisk], 0)
	assert.Len(alloc[fwdp.RoleFwd], 7)
	lcInput, lcOutput := alloc[fwdp.RoleInput].ByNumaSocket(), alloc[fwdp.RoleOutput].ByNumaSocket()
	assert.GreaterOrEqual(len(lcInput[numa0]), 1)
	assert.GreaterOrEqual(len(lcInput[numa1]), 1)
	assert.GreaterOrEqual(len(lcOutput[numa0]), 1)
	assert.GreaterOrEqual(len(lcOutput[numa1]), 1)
}
