package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
)

func TestPciAddress(t *testing.T) {
	assert, _ := makeAR(t)

	a, e := dpdk.ParsePciAddress("0000:8F:00.0")
	assert.NoError(e)
	assert.Equal("0000:8f:00.0", a.String())
	assert.Equal("8f:00.0", a.ShortString())

	a, e = dpdk.ParsePciAddress("01:00.0")
	assert.NoError(e)
	assert.Equal("0000:01:00.0", a.String())
	assert.Equal("01:00.0", a.ShortString())

	_, e = dpdk.ParsePciAddress("bad")
	assert.Error(e)
}
