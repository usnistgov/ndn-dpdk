package gqlclient_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
)

func TestClient(t *testing.T) {
	assert, require := makeAR(t)

	c, e := gqlclient.New(serverURI)
	require.NoError(e)

	var reply struct {
		Version string `json:"version"`
	}
	e = c.Do(`
		query {
			version {
				version
			}
		}
	`, nil, "version", &reply)
	assert.NoError(e)
	assert.NotZero(reply.Version)
}
