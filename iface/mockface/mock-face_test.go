package mockface_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal/ealtestenv"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/iface/mockface"
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
