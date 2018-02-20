package fib_test

import (
	"testing"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func createFib() *fib.Fib {
	cfg := fib.Config{
		Id:         "TestFib",
		MaxEntries: 255,
		NBuckets:   64,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
		StartDepth: 2,
	}

	fib, e := fib.New(cfg)
	if e != nil {
		panic(e)
	}
	return fib
}

func makeFibEntry(nameStr string, nexthops ...iface.FaceId) (entry *fib.Entry) {
	entry = new(fib.Entry)
	name, _ := ndn.ParseName(nameStr)
	entry.SetName(name)
	entry.SetNexthops(nexthops)
	return entry
}

func TestFibInsertErase(t *testing.T) {
	assert, require := makeAR(t)

	fib := createFib()
	defer fib.Close()
	mp := fib.GetMempool()
	assert.Zero(fib.Len())
	assert.Zero(mp.CountInUse())
	nameA, _ := ndn.ParseName("/A")
	assert.Nil(fib.Find(nameA))

	_, e := fib.Insert(makeFibEntry("/A"))
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
	assert.True(mp.CountInUse() >= 1)

	assert.NotNil(fib.Find(nameA))
	names := fib.ListNames()
	require.Len(names, 1)
	assert.Equal(nameA, names[0])

	assert.NoError(fib.Erase(nameA))
	assert.Zero(fib.Len())
	assert.Nil(fib.Find(nameA))
	assert.Len(fib.ListNames(), 0)
	assert.Error(fib.Erase(nameA))
	assert.Zero(fib.Len())

	urcu.Barrier()
	assert.Zero(mp.CountInUse())

	fib.Insert(makeFibEntry("/A", 2886))
	fib.Insert(makeFibEntry("/A/B/C", 1916))
	fib.Insert(makeFibEntry("/E/F/G/H", 7505))
	fib.Insert(makeFibEntry("/E/F", 2143))
	assert.Equal(4, fib.Len())
	assert.Equal(1, fib.CountVirtuals())
	assert.Len(fib.ListNames(), 4)
}

func TestFibLpm(t *testing.T) {
	assert, _ := makeAR(t)

	fib := createFib()
	defer fib.Close()

	lpm := func(nameStr string) int {
		name, _ := ndn.ParseName(nameStr)
		entry := fib.Lpm(name)
		if entry == nil {
			return 0
		}
		return int(entry.GetNexthops()[0])
	}

	fib.Insert(makeFibEntry("/", 5000))
	fib.Insert(makeFibEntry("/A", 5001))
	fib.Insert(makeFibEntry("/A/B/C", 5002))
	assert.Len(fib.ListNames(), 3)

	assert.Equal(5000, lpm("/"))
	assert.Equal(5001, lpm("/A"))
	assert.Equal(5000, lpm("/AB"))
	assert.Equal(5001, lpm("/A/B"))
	assert.Equal(5002, lpm("/A/B/C"))
	assert.Equal(5002, lpm("/A/B/C/D"))
	assert.Equal(5001, lpm("/A/B/CD"))

	emptyName, _ := ndn.ParseName("/")
	fib.Erase(emptyName)
	assert.Equal(0, lpm("/"))
	assert.Equal(5001, lpm("/A"))
	assert.Equal(0, lpm("/AB"))

	fib.Insert(makeFibEntry("/E/F/G/H", 7505))
	fib.Insert(makeFibEntry("/E/F", 2143))
	assert.Equal(7505, lpm("/E/F/G/H/I/J"))
	assert.Equal(2143, lpm("/E/F/G/K"))
}
