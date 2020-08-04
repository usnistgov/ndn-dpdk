package gqlclient_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/app/version"
	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
)

func TestClient(t *testing.T) {
	assert, require := makeAR(t)

	c, e := gqlclient.New(serverURI)
	require.NoError(e)

	var reply string
	e = c.Do(`
		query {
			version
		}
	`, nil, "version", &reply)
	assert.NoError(e)
	assert.Equal(version.COMMIT, reply)
}
