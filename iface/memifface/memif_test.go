package memifface_test

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/iface/memifface"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"go4.org/must"
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

	helper := exec.Command(os.Args[0], memifbridgeArg, socketName)
	helperIn, e := helper.StdinPipe()
	require.NoError(e)
	helper.Stdout = os.Stdout
	helper.Stderr = os.Stderr
	require.NoError(helper.Start())
	defer helper.Process.Kill()
	time.Sleep(1 * time.Second)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()

	var st unix.Stat_t
	require.NoError(unix.Stat(socketName, &st))
	assert.EqualValues(0, st.Uid)
	assert.EqualValues(8000, st.Gid)

	helperIn.Write([]byte("."))
	assert.NoError(helper.Wait())
}

const memifbridgeArg = "memifbridge"

func memifbridgeHelper() {
	socketName := os.Args[2]
	locA := memiftransport.Locator{
		Role:       memiftransport.RoleClient,
		SocketName: socketName,
		ID:         7655,
	}
	locB := memiftransport.Locator{
		Role:       memiftransport.RoleClient,
		SocketName: socketName,
		ID:         1891,
	}

	bridge, e := memiftransport.NewBridge(locA, locB)
	if e != nil {
		panic(e)
	}

	io.ReadAtLeast(os.Stdin, make([]byte, 1), 1)
	must.Close(bridge)
}
