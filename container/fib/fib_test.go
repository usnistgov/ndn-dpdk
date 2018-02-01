package fib_test

import (
	"testing"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/core/urcu"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func createFib() *fib.Fib {
	cfg := fib.Config{
		Id:         "TestFib",
		MaxEntries: 255,
		NBuckets:   64,
		NumaSocket: dpdk.NUMA_SOCKET_ANY,
	}

	fib, e := fib.New(cfg)
	if e != nil {
		panic(e)
	}
	return fib
}

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

func TestFibInsertErase(t *testing.T) {
	assert, _ := makeAR(t)
	rcuRs := urcu.NewReadSide()
	defer rcuRs.Close()

	fib := createFib()
	defer fib.Close()
	mp := fib.GetMempool()
	assert.Zero(fib.Len())
	assert.Zero(mp.CountInUse())

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

func TestFibLpm(t *testing.T) {
	assert, require := makeAR(t)
	rcuRs := urcu.NewReadSide()
	defer rcuRs.Close()

	fib := createFib()
	defer fib.Close()

	lpm := func(nameStr string) int {
		tb, e := ndn.EncodeNameFromUri(nameStr)
		require.NoError(e)
		pkt := dpdktestenv.PacketFromBytes(tb)
		defer pkt.Close()
		d := ndn.NewTlvDecoder(pkt)
		name, e := d.ReadName()
		require.NoError(e, nameStr)

		entry := fib.Lpm(&name, rcuRs)
		if entry == nil {
			return 0
		}
		return int(entry.GetNexthops()[0])
	}

	fib.Insert(makeFibEntry("/", 5000))
	fib.Insert(makeFibEntry("/A", 5001))
	fib.Insert(makeFibEntry("/A/B/C", 5002))

	assert.Equal(5000, lpm("/"))
	assert.Equal(5001, lpm("/A"))
	assert.Equal(5000, lpm("/AB"))
	assert.Equal(5001, lpm("/A/B"))
	assert.Equal(5002, lpm("/A/B/C"))
	assert.Equal(5002, lpm("/A/B/C/D"))
	assert.Equal(5001, lpm("/A/B/CD"))

	fib.Erase(ndn.TlvBytes{}) // erase '/' entry
	assert.Equal(0, lpm("/"))
	assert.Equal(5001, lpm("/A"))
	assert.Equal(0, lpm("/AB"))
}
