package ifacetestenv

import (
	"encoding/json"
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// CheckLocatorMarshal checks JSON marshaling of the Locator.
func CheckLocatorMarshal(t testing.TB, loc iface.Locator) {
	assert, _ := testenv.MakeAR(t)
	var locw iface.LocatorWrapper
	locw.Locator = loc

	jsonEncoded, e := json.Marshal(locw)
	if assert.NoError(e) {
		var jsonDecoded iface.LocatorWrapper
		assert.NoError(json.Unmarshal(jsonEncoded, &jsonDecoded), "%s", jsonEncoded)
	}
}
