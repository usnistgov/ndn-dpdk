package mockface_test

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/mockface"
)

func TestMockFace(t *testing.T) {
	dpdktestenv.InitEal()
	assert, _ := dpdktestenv.MakeAR(t)

	_, mockface.FaceMempools = ifacetestfixture.MakeMempools()

	face := mockface.New()
	defer face.Close()

	loc := face.GetLocator().(mockface.Locator)
	assert.Equal("mock", loc.Scheme)
	ifacetestfixture.CheckLocatorMarshal(t, loc)
}
