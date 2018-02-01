package fib_test

import (
	"testing"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func makeFibEntry(name string, nexthops ...iface.FaceId) (entry *fib.Entry) {
	entry = new(fib.Entry)
	comps, e := ndn.EncodeNameComponentsFromUri(name)
	if e != nil {
		panic(e)
	}
	entry.SetName(comps)
	entry.SetNexthops(nexthops)
	return entry
}

func TestFib(t *testing.T) {
	assert, require := makeAR(t)
	rcuRs := urcu.NewReadSide()
	defer rcuRs.Close()

	cfg := fib.Config{
		Id:         "TestFib",
		MaxEntries: 255,
		NBuckets:   64,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}

	fib, e := fib.New(cfg)
	require.NoError(e)
	defer fib.Close()
	mp := fib.GetMempool()
	assert.Zero(fib.Len())
	assert.Zero(mp.CountInUse())

	_, e = fib.Insert(makeFibEntry("/A"))
	assert.Error(e) // cannot insert: entry has no nexthop
	assert.Zero(mp.CountInUse())

	isNew, e := fib.Insert(makeFibEntry("/A", 4076))
	assert.NoError(e)
	assert.True(isNew)
	assert.Equal(1, fib.Len())
	assert.Equal(1, mp.CountInUse())

	isNew, e = fib.Insert(makeFibEntry("/A", 3092))
	assert.NoError(e)
	assert.False(isNew)
	assert.Equal(1, fib.Len())
	assert.Equal(2, mp.CountInUse())

	nameA, _ := ndn.EncodeNameComponentsFromUri("/A")
	assert.True(fib.Erase(nameA))
	assert.Zero(fib.Len())
	assert.False(fib.Erase(nameA))
	assert.Zero(fib.Len())
	assert.Equal(2, mp.CountInUse())

	rcuRs.Quiescent()
	urcu.Barrier()
	assert.Equal(0, mp.CountInUse())
}
