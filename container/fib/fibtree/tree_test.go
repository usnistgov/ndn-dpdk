package fibtree_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtree"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestInsertErase(testingT *testing.T) {
	assert, _ := makeAR(testingT)
	t := fibtree.New(2)
	assert.Equal(0, t.CountEntries())
	assert.Equal(1, t.CountNodes())

	u := t.Insert(makeEntry("/%00/A/B/B", 1, 1))
	if real := u.Real(); assert.NotNil(real) {
		nameEqual(assert, "/%00/A/B/B", real)
		assert.Equal(fibdef.ActInsert, real.Action)
		assert.Nil(real.WithVirt)
	}
	if virt := u.Virt(); assert.NotNil(virt) {
		nameEqual(assert, "/%00/A", virt)
		assert.Equal(fibdef.ActInsert, virt.Action)
		assert.False(virt.HasReal)
		assert.Equal(2, virt.Height)
	}
	assert.Equal(1, t.CountEntries())
	assert.Equal(5, t.CountNodes())

	u = t.Insert(makeEntry("/%00/A/B/B", 1, 1))
	assert.Nil(u.Real())
	assert.Nil(u.Virt())
	assert.Equal(1, t.CountEntries())
	assert.Equal(5, t.CountNodes())

	u = t.Insert(makeEntry("/%00/A/B/B", 1, 1, 2))
	if real := u.Real(); assert.NotNil(real) {
		assert.Equal(fibdef.ActReplace, real.Action)
	}
	assert.Nil(u.Virt())
	assert.Equal(1, t.CountEntries())
	assert.Equal(5, t.CountNodes())

	u = t.Insert(makeEntry("/%00/A/C", 1, 1))
	if real := u.Real(); assert.NotNil(real) {
		nameEqual(assert, "/%00/A/C", real)
		assert.Equal(fibdef.ActInsert, real.Action)
		assert.Nil(real.WithVirt)
	}
	assert.Nil(u.Virt())
	assert.Equal(2, t.CountEntries())
	assert.Equal(6, t.CountNodes())

	u = t.Erase(ndn.ParseName("/%00/A/B/B"))
	if real := u.Real(); assert.NotNil(real) {
		nameEqual(assert, "/%00/A/B/B", real)
		assert.Equal(fibdef.ActErase, real.Action)
		assert.Nil(real.WithVirt)
	}
	if virt := u.Virt(); assert.NotNil(virt) {
		nameEqual(assert, "/%00/A", virt)
		assert.Equal(fibdef.ActReplace, virt.Action)
		assert.False(virt.HasReal)
		assert.Equal(1, virt.Height)
	}
	assert.Equal(1, t.CountEntries())
	assert.Equal(4, t.CountNodes())

	u = t.Insert(makeEntry("/%00/A", 1, 1))
	if real := u.Real(); assert.NotNil(real) {
		nameEqual(assert, "/%00/A", real)
		assert.Equal(fibdef.ActInsert, real.Action)
		if assert.NotNil(real.WithVirt) {
			nameEqual(assert, "/%00/A", real.WithVirt)
			assert.Equal(fibdef.ActReplace, real.WithVirt.Action)
			assert.Equal(1, real.WithVirt.Height)
		}
	}
	assert.Nil(u.Virt())
	assert.Equal(2, t.CountEntries())
	assert.Equal(4, t.CountNodes())

	u = t.Erase(ndn.ParseName("/%00/A/B/B"))
	assert.Nil(u.Real())
	assert.Nil(u.Virt())
	assert.Equal(2, t.CountEntries())
	assert.Equal(4, t.CountNodes())

	u = t.Erase(ndn.ParseName("/%00"))
	assert.Nil(u.Real())
	assert.Nil(u.Virt())
	assert.Equal(2, t.CountEntries())
	assert.Equal(4, t.CountNodes())

	u = t.Erase(ndn.ParseName("/%00/A/C"))
	if real := u.Real(); assert.NotNil(real) {
		nameEqual(assert, "/%00/A/C", real)
		assert.Equal(fibdef.ActErase, real.Action)
		assert.Nil(real.WithVirt)
	}
	if virt := u.Virt(); assert.NotNil(virt) {
		nameEqual(assert, "/%00/A", virt)
		assert.Equal(fibdef.ActErase, virt.Action)
		assert.True(virt.HasReal)
	}
	assert.Equal(1, t.CountEntries())
	assert.Equal(3, t.CountNodes())

	u = t.Erase(ndn.ParseName("/%00/A"))
	if real := u.Real(); assert.NotNil(real) {
		nameEqual(assert, "/%00/A", real)
		assert.Equal(fibdef.ActErase, real.Action)
		assert.Nil(real.WithVirt)
	}
	assert.Nil(u.Virt())
	assert.Equal(0, t.CountEntries())
	assert.Equal(1, t.CountNodes())
}
