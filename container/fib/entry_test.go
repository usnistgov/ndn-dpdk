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
	assert.Len(entry.GetName(), 0)
	assert.Equal(0, entry.GetNComps())
	assert.Len(entry.GetNexthops(), 0)

	name, _ := ndn.EncodeNameComponentsFromUri("/A/B")
	assert.NoError(entry.SetName(name))
	assert.Equal(name, entry.GetName())
	assert.Equal(2, entry.GetNComps())

	nexthops := []iface.FaceId{2302, 1067, 1122}
	assert.NoError(entry.SetNexthops(nexthops))
	assert.Equal(nexthops, entry.GetNexthops())

	name2 := name
	for len(name2) <= fib.MAX_NAME_LEN {
		name2 = append(name2, name...)
	}
	assert.Error(entry.SetName(name2))

	nexthops2 := make([]iface.FaceId, 0)
	for len(nexthops2) <= fib.MAX_NEXTHOPS {
		nexthops2 = append(nexthops2, iface.FaceId(5000+len(nexthops2)))
	}
	assert.Error(entry.SetNexthops(nexthops2))
}
