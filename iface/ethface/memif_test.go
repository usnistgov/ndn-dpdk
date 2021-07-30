package ethface_test

import (
	"io"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"go4.org/must"
)

func TestMemif(t *testing.T) {
	assert, require := makeAR(t)

	socketName, del := testenv.TempName("memif.sock")
	defer del()

	fixture := ifacetestenv.NewFixture(t)
	defer fixture.Close()

	var locA ethface.MemifLocator
	locA.SocketName = socketName
	locA.ID = 7655
	faceA, e := locA.CreateFace()
	require.NoError(e)
	assert.Equal("memif", faceA.Locator().Scheme())

	var locB ethface.MemifLocator
	locB.SocketName = socketName
	locB.ID = 1891
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

	helperIn.Write([]byte("."))
	assert.NoError(helper.Wait())
}

const memifbridgeArg = "memifbridge"

func memifbridgeHelper() {
	socketName := os.Args[2]
	var locA, locB memiftransport.Locator
	locA.SocketName = socketName
	locA.ID = 7655
	locB.SocketName = socketName
	locB.ID = 1891

	bridge, e := memiftransport.NewBridge(locA, locB, memiftransport.RoleClient)
	if e != nil {
		panic(e)
	}

	io.ReadAtLeast(os.Stdin, make([]byte, 1), 1)
	must.Close(bridge)
}
