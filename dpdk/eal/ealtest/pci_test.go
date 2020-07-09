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

	a, e = eal.ParsePciAddress("01:00.0")
	assert.NoError(e)
	assert.Equal("0000:01:00.0", a.String())

	_, e = eal.ParsePciAddress("bad")
	assert.Error(e)

	a.Bus, a.Slot, a.Function = "5e", "01", "0"
	assert.Equal(`"0000:5e:01.0"`, toJSON(a))

	var decoded eal.PciAddress
	fromJSON(`"0000:5e:01.0"`, &decoded)
	assert.Equal(a, decoded)
}
