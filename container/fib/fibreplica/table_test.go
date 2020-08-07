package fibreplica_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/container/fib/fibdef"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibreplica"
	"github.com/usnistgov/ndn-dpdk/container/fib/fibtree"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestInsertErase(testingT *testing.T) {
	assert, require := makeAR(testingT)
	tree := fibtree.New(2)
	t, e := fibreplica.New(fibdef.Config{
		Capacity:   1023,
		StartDepth: 2,
	}, 1, eal.NumaSocket{})
	require.NoError(e)
	defer t.Close()

	do := func(tu fibdef.Update) {
		u, e := t.PrepareUpdate(tu)
		require.NoError(e)
		t.ExecuteUpdate(u)
		tu.Commit()
	}

	// insert above startDepth
	do(tree.Insert(makeEntry("/A", nil, 1)))
	if entry := t.Get(ndn.ParseName("/A")); assert.NotNil(entry) {
		assert.False(entry.IsVirt())
	}

	// insert below startDepth
	do(tree.Insert(makeEntry("/A/B/C", nil, 1)))
	if entry := t.Get(ndn.ParseName("/A/B/C")); assert.NotNil(entry) {
		assert.False(entry.IsVirt())
	}
	if entry := t.Get(ndn.ParseName("/A/B")); assert.NotNil(entry) {
		assert.True(entry.IsVirt())
		assert.Nil(entry.Real())
	}

	// insert at startDepth
	do(tree.Insert(makeEntry("/A/B", nil, 1)))
	if entry := t.Get(ndn.ParseName("/A/B")); assert.NotNil(entry) {
		assert.True(entry.IsVirt())
		if realEntry := entry.Real(); assert.NotNil(realEntry) {
			assert.False(realEntry.IsVirt())
		}
	}

	// delete below startDepth
	do(tree.Erase(ndn.ParseName("/A/B/C")))
	assert.Nil(t.Get(ndn.ParseName("/A/B/C")))
	if entry := t.Get(ndn.ParseName("/A/B")); assert.NotNil(entry) {
		assert.False(entry.IsVirt())
	}

	// insert below startDepth
	do(tree.Insert(makeEntry("/A/B/D", nil, 1)))
	if entry := t.Get(ndn.ParseName("/A/B/D")); assert.NotNil(entry) {
		assert.False(entry.IsVirt())
	}
	if entry := t.Get(ndn.ParseName("/A/B")); assert.NotNil(entry) {
		assert.True(entry.IsVirt())
		if realEntry := entry.Real(); assert.NotNil(realEntry) {
			assert.False(realEntry.IsVirt())
		}
	}

	// delete at startDepth
	do(tree.Erase(ndn.ParseName("/A/B")))
	if entry := t.Get(ndn.ParseName("/A/B")); assert.NotNil(entry) {
		assert.True(entry.IsVirt())
		assert.Nil(entry.Real())
	}

	// delete below startDepth
	do(tree.Erase(ndn.ParseName("/A/B/D")))
	assert.Nil(t.Get(ndn.ParseName("/A/B/D")))
	assert.Nil(t.Get(ndn.ParseName("/A/B")))
}
