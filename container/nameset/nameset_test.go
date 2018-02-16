package nameset_test

import (
	"testing"

	"ndn-dpdk/container/nameset"
	"ndn-dpdk/ndn"
)

func TestNameSet(t *testing.T) {
	assert, _ := makeAR(t)

	nameEmpty, _ := ndn.ParseName("/")
	nameA, _ := ndn.ParseName("/A")
	nameAB, _ := ndn.ParseName("/A/B")
	nameABC, _ := ndn.ParseName("/A/B/C")
	nameCD, _ := ndn.ParseName("/C/D")

	set := nameset.New()
	defer set.Close()
	assert.Equal(0, set.Len())
	assert.Equal(-1, set.FindExact(nameEmpty))
	assert.Equal(-1, set.FindPrefix(nameEmpty))

	set.InsertWithZeroUsr(nameAB, 8)
	assert.Equal(1, set.Len())
	assert.True(set.FindPrefix(nameA) < 0)
	indexAB := set.FindExact(nameAB)
	if assert.True(indexAB >= 0) {
		assert.True(nameAB.Equal(set.GetName(indexAB)))
		assert.True(set.GetUsr(indexAB) != nil)
	}
	assert.True(set.FindPrefix(nameAB) >= 0)
	assert.True(set.FindPrefix(nameABC) >= 0)

	set.Insert(nameCD)
	assert.Equal(2, set.Len())
	assert.True(set.FindExact(nameAB) >= 0)
	assert.True(set.FindExact(nameCD) >= 0)

	set.Erase(set.FindExact(nameCD))
	assert.Equal(1, set.Len())
	assert.True(set.FindExact(nameAB) >= 0)
	assert.True(set.FindExact(nameCD) < 0)
}
