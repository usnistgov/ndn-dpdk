package ealtest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func TestPciAddress(t *testing.T) {
	assert, _ := makeAR(t)

	a, e := eal.ParsePciAddress("0000:8F:00.0")
	assert.NoError(e)
	assert.Equal("0000:8f:00.0", a.String())
	assert.Equal("8f:00.0", a.ShortString())

	a, e = eal.ParsePciAddress("01:00.0")
	assert.NoError(e)
	assert.Equal("0000:01:00.0", a.String())
	assert.Equal("01:00.0", a.ShortString())

	_, e = eal.ParsePciAddress("bad")
	assert.Error(e)
}
