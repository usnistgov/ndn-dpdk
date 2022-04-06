package pciaddr_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/pciaddr"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
)

var (
	makeAR = testenv.MakeAR
)

func TestPCIAddress(t *testing.T) {
	assert, _ := makeAR(t)

	a, e := pciaddr.Parse("0000:8F:00.0")
	assert.NoError(e)
	assert.Equal("0000:8f:00.0", a.String())

	a, e = pciaddr.Parse("01:00.0")
	assert.NoError(e)
	assert.Equal("0000:01:00.0", a.String())

	_, e = pciaddr.Parse("bad")
	assert.Error(e)

	a.Bus, a.Slot, a.Function = 0x5e, 0x01, 0x0
	assert.Equal(`"0000:5e:01.0"`, testenv.ToJSON(a))

	assert.Equal(a, testenv.FromJSON[pciaddr.PCIAddress](`"0000:5e:01.0"`))
}
