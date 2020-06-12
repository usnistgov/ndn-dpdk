package mockface_test

import (
	"testing"

	"ndn-dpdk/core/testenv"
	"ndn-dpdk/dpdk/eal/ealtestenv"
	"ndn-dpdk/iface/ifacetestenv"
	"ndn-dpdk/iface/mockface"
)

func TestMockFace(t *testing.T) {
	ealtestenv.InitEal()
	assert, _ := testenv.MakeAR(t)

	face := mockface.New()
	defer face.Close()

	loc := face.GetLocator().(mockface.Locator)
	assert.Equal("mock", loc.Scheme)
	ifacetestenv.CheckLocatorMarshal(t, loc)
}
