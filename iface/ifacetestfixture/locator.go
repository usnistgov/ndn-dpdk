package ifacetestfixture

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v2"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/iface"
)

func CheckLocatorMarshal(t *testing.T, loc iface.Locator) {
	assert, _ := dpdktestenv.MakeAR(t)

	jsonEncoded, e := json.Marshal(loc)
	if assert.NoError(e) {
		var jsonDecoded iface.LocatorWrapper
		assert.NoError(json.Unmarshal(jsonEncoded, &jsonDecoded), "%s", jsonEncoded)
	}

	yamlEncoded, e := yaml.Marshal(loc)
	if assert.NoError(e) {
		var yamlDecoded iface.LocatorWrapper
		assert.NoError(yaml.Unmarshal(yamlEncoded, &yamlDecoded), "%s", yamlEncoded)
	}
}
