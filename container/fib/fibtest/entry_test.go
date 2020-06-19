package fibtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestEntry(t *testing.T) {
	assert, _ := makeAR(t)

	var entry fib.Entry
	ndntestenv.NameEqual(assert, "/", entry.GetName())
	assert.Equal(0, entry.CountComps())
	assert.Len(entry.GetNexthops(), 0)

	name := ndn.ParseName("/A/B")
	assert.NoError(entry.SetName(name))
	assert.Zero(name.Compare(entry.GetName()))
	assert.Equal(2, entry.CountComps())

	nexthops := []iface.FaceId{2302, 1067, 1122}
	assert.NoError(entry.SetNexthops(nexthops))
	assert.Equal(nexthops, entry.GetNexthops())

	name2V, _ := name.MarshalBinary()
	for len(name2V) <= fib.MAX_NAME_LEN {
		name2V = append(name2V, name2V...)
	}

	var name2 ndn.Name
	name2.UnmarshalBinary(name2V)
	assert.Error(entry.SetName(name2))

	nexthops2 := make([]iface.FaceId, 0)
	for len(nexthops2) <= fib.MAX_NEXTHOPS {
		nexthops2 = append(nexthops2, iface.FaceId(5000+len(nexthops2)))
	}
	assert.Error(entry.SetNexthops(nexthops2))
}
