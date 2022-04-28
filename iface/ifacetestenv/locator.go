package ifacetestenv

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// CheckLocatorMarshal checks JSON marshaling of a Locator.
func CheckLocatorMarshal(t testing.TB, loc iface.Locator) {
	assert, _ := testenv.MakeAR(t)
	var decoded iface.LocatorWrapper
	assert.NoError(jsonhelper.Roundtrip(iface.LocatorWrapper{Locator: loc}, &decoded, jsonhelper.DisallowUnknownFields))
}
