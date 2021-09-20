package memiftransport_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

func TestCoexist(t *testing.T) {
	assert, _ := makeAR(t)

	locAc0 := memiftransport.Locator{
		Role:       memiftransport.RoleClient,
		SocketName: "/tmp/memifA.sock",
		ID:         0,
	}
	locAc1 := memiftransport.Locator{
		Role:       memiftransport.RoleClient,
		SocketName: "/tmp/memifA.sock",
		ID:         1,
	}
	locAs0 := memiftransport.Locator{
		Role:       memiftransport.RoleServer,
		SocketName: "/tmp/memifA.sock",
		ID:         0,
	}
	locBs0 := memiftransport.Locator{
		Role:       memiftransport.RoleServer,
		SocketName: "/tmp/memifB.sock",
		ID:         0,
	}

	c := memiftransport.NewCoexistMap()
	assert.False(c.Has(locAc0.SocketName))
	assert.NoError(c.Check(locAc0))
	c.Add(locAc0)
	assert.True(c.Has(locAc0.SocketName))
	assert.Error(c.Check(locAc0))
	assert.NoError(c.Check(locAc1))
	c.Add(locAc1)
	assert.Error(c.Check(locAs0))
	assert.NoError(c.Check(locBs0))
	c.Add(locBs0)
	c.Remove(locAc1)
	c.Remove(locAc0)
	assert.NoError(c.Check(locAs0))
	c.Add(locAs0)
}
