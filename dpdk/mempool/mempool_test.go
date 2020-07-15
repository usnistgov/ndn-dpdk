package mempool_test

import (
	"testing"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
)

func TestMempool(t *testing.T) {
	assert, require := makeAR(t)

	mp, e := mempool.New(mempool.Config{
		Capacity:    63,
		ElementSize: 256,
	})
	require.NoError(e)
	require.NotNil(mp)
	defer mp.Close()

	assert.Equal(256, mp.SizeofElement())
	assert.Equal(63, mp.CountAvailable())
	assert.Equal(0, mp.CountInUse())

	var objs [64]unsafe.Pointer
	e = mp.Alloc(objs[:])
	assert.Error(e)
	assert.Equal(63, mp.CountAvailable())
	assert.Equal(0, mp.CountInUse())

	e = mp.Alloc(objs[:30])
	assert.NoError(e)
	assert.Equal(33, mp.CountAvailable())
	assert.Equal(30, mp.CountInUse())

	mp.Free(objs[0:1])
	assert.Equal(34, mp.CountAvailable())
	assert.Equal(29, mp.CountInUse())

	mp.Free(objs[1:30])
	assert.Equal(63, mp.CountAvailable())
}
