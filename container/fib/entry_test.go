package fib_test

import (
	"testing"

	"ndn-dpdk/container/fib"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

func TestEntry(t *testing.T) {
	assert, _ := makeAR(t)

	var entry fib.Entry
	assert.Equal("/", entry.GetName().String())
	assert.Equal(0, entry.GetNComps())
	assert.Len(entry.GetNexthops(), 0)

	name, _ := ndn.ParseName("/A/B")
	assert.NoError(entry.SetName(name))
	assert.True(name.Equal(entry.GetName()))
	assert.Equal(2, entry.GetNComps())

	nexthops := []iface.FaceId{2302, 1067, 1122}
	assert.NoError(entry.SetNexthops(nexthops))
	assert.Equal(nexthops, entry.GetNexthops())

	name2V := name.GetValue()
	for len(name2V) <= fib.MAX_NAME_LEN {
		name2V = append(name2V, name2V...)
	}
	name2, _ := ndn.NewName(name2V)
	assert.Error(entry.SetName(name2))

	nexthops2 := make([]iface.FaceId, 0)
	for len(nexthops2) <= fib.MAX_NEXTHOPS {
		nexthops2 = append(nexthops2, iface.FaceId(5000+len(nexthops2)))
	}
	assert.Error(entry.SetNexthops(nexthops2))
}
