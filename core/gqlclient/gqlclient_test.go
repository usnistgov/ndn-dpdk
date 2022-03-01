package gqlclient_test

import (
	"context"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/gqlclient"
)

func TestClient(t *testing.T) {
	assert, require := makeAR(t)

	c, e := gqlclient.New(serverConfig)
	require.NoError(e)
	defer c.Close()

	var reply struct {
		Version string `json:"version"`
	}
	e = c.Do(context.Background(), `
		query {
			version {
				version
			}
		}
	`, nil, "version", &reply)
	assert.NoError(e)
	assert.NotZero(reply.Version)

	ticks := make(chan int, 20)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	e = c.Subscribe(ctx, `
		subscription tick($interval: NNNanoseconds!) {
			tick(interval: $interval)
		}
	`, map[string]interface{}{
		"interval": "200ms",
	}, "tick", ticks)
	assert.NoError(e)
	assert.Greater(len(ticks), 5)
}
