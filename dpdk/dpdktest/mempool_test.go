package dpdktest

import (
	"testing"
	"unsafe"

	"ndn-dpdk/dpdk"
)

func TestMempool(t *testing.T) {
	assert, require := makeAR(t)

	mp, e := dpdk.NewMempool("MP", 63, 256, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	defer mp.Close()

	assert.Equal(256, mp.SizeofElement())
	assert.Equal(63, mp.CountAvailable())
	assert.Equal(0, mp.CountInUse())

	var objs [63]unsafe.Pointer
	e = mp.AllocBulk(objs[30:])
	assert.NoError(e)
	assert.Equal(30, mp.CountAvailable())
	assert.Equal(33, mp.CountInUse())
	for i := 0; i < 30; i++ {
		objs[i] = mp.Alloc()
		assert.False(objs[i] == nil)
	}
	assert.Equal(0, mp.CountAvailable())
	assert.Equal(63, mp.CountInUse())
	assert.True(mp.Alloc() == nil)
	mp.Free(objs[0])
	assert.Equal(1, mp.CountAvailable())
	assert.Equal(62, mp.CountInUse())

	for i := 1; i < 30; i++ {
		mp.Free(objs[i])
	}
	mp.FreeBulk(objs[30:])
	assert.Equal(63, mp.CountAvailable())
}

func TestPktmbufPool(t *testing.T) {
	assert, require := makeAR(t)

	mp, e := dpdk.NewPktmbufPool("MP", 63, 0, 1000, dpdk.NUMA_SOCKET_ANY)
	require.NoError(e)
	defer mp.Close()

	assert.Equal(63, mp.CountAvailable())
	assert.Equal(0, mp.CountInUse())

	var mbufs [63]dpdk.Mbuf
	e = mp.AllocBulk(mbufs[30:])
	assert.NoError(e)
	assert.Equal(30, mp.CountAvailable())
	assert.Equal(33, mp.CountInUse())
	for i := 0; i < 30; i++ {
		mbufs[i], e = mp.Alloc()
		assert.NoError(e)
	}
	assert.Equal(0, mp.CountAvailable())
	assert.Equal(63, mp.CountInUse())
	_, e = mp.Alloc()
	assert.Error(e)
	mbufs[0].Close()
	assert.Equal(1, mp.CountAvailable())
	assert.Equal(62, mp.CountInUse())

	for i := 1; i < 63; i++ {
		mbufs[i].Close()
	}
	assert.Equal(63, mp.CountAvailable())
}
