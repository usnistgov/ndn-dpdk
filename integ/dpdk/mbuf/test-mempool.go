package main

import (
	"github.com/stretchr/testify/assert"
	"ndn-traffic-dpdk/dpdk"
)

func testMempool() {
	assert := assert.New(t)

	assert.EqualValues(63, mp.CountAvailable())
	assert.EqualValues(0, mp.CountInUse())

	var mbufs [63]dpdk.Mbuf
	e := mp.AllocBulk(mbufs[30:])
	assert.NoError(e)
	assert.EqualValues(30, mp.CountAvailable())
	assert.EqualValues(33, mp.CountInUse())
	for i := 0; i < 30; i++ {
		mbufs[i], e = mp.Alloc()
		assert.NoError(e)
	}
	assert.EqualValues(0, mp.CountAvailable())
	assert.EqualValues(63, mp.CountInUse())
	_, e = mp.Alloc()
	assert.Error(e)
	mbufs[0].Close()
	assert.EqualValues(1, mp.CountAvailable())
	assert.EqualValues(62, mp.CountInUse())

	for i := 1; i < 63; i++ {
		mbufs[i].Close()
	}
	assert.EqualValues(63, mp.CountAvailable())
}
