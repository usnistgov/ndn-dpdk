package ifacetestfixture

import (
	"encoding/json"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
)

func CheckLocatorMarshal(t *testing.T, loc iface.Locator) {
	assert, _ := dpdktestenv.MakeAR(t)
	locw := iface.LocatorWrapper{loc}

	jsonEncoded, e := json.Marshal(locw)
	if assert.NoError(e) {
		var jsonDecoded iface.LocatorWrapper
		assert.NoError(json.Unmarshal(jsonEncoded, &jsonDecoded), "%s", jsonEncoded)
	}
}
