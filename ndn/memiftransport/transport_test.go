package memiftransport_test

import (
	"path"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestTransport(t *testing.T) {
	assert, require := makeAR(t)
	dir := t.TempDir()

	require.NoError(memiftransport.ForkBridgeHelper(memiftransport.Locator{
		Role:       memiftransport.RoleServer,
		SocketName: path.Join(dir, "memifA.sock"),
		ID:         1216,
	}, memiftransport.Locator{
		Role:       memiftransport.RoleServer,
		SocketName: path.Join(dir, "memifB.sock"),
		ID:         2643,
	}, func() {
		trA, e := memiftransport.New(memiftransport.Locator{
			SocketName: path.Join(dir, "memifA.sock"),
			ID:         1216,
		})
		require.NoError(e)
		assert.Equal(memiftransport.DefaultDataroom, trA.MTU())
		trB, e := memiftransport.New(memiftransport.Locator{
			SocketName: path.Join(dir, "memifB.sock"),
			ID:         2643,
		})
		require.NoError(e)

		var c ndntestenv.L3FaceTester
		c.CheckTransport(t, trA, trB)
	}))
}
