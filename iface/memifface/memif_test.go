package memifface_test

import (
	"path/filepath"
	"testing"

	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/iface/memifface"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"golang.org/x/sys/unix"
)

func TestMemif(t *testing.T) {
	assert, require := makeAR(t)
	fixture := ifacetestenv.NewFixture(t)
	socketName := filepath.Join(t.TempDir(), "subdir/memif.sock")

	var locA memifface.Locator
	locA.SocketName = socketName
	locA.ID = 7655
	locA.SocketOwner = &[2]int{0, 8000}
	faceA, e := locA.CreateFace()
	require.NoError(e)
	assert.Equal("memif", faceA.Locator().Scheme())

	var locB memifface.Locator
	locB.SocketName = socketName
	locB.ID = 1891
	locB.SocketOwner = &[2]int{0, 8001}
	faceB, e := locB.CreateFace()
	require.NoError(e)

	require.NoError(memiftransport.ForkBridgeHelper(memiftransport.Locator{
		Role:       memiftransport.RoleClient,
		SocketName: socketName,
		ID:         7655,
	}, memiftransport.Locator{
		Role:       memiftransport.RoleClient,
		SocketName: socketName,
		ID:         1891,
	}, func() {
		fixture.RunTest(faceA, faceB)
		fixture.CheckCounters()

		var st unix.Stat_t
		require.NoError(unix.Stat(socketName, &st))
		assert.EqualValues(0, st.Uid)
		assert.EqualValues(8000, st.Gid)
	}))
}
