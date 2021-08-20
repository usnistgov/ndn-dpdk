package memiftransport_test

import (
	"io"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestTransport(t *testing.T) {
	assert, require := makeAR(t)

	dir, del := testenv.TempDir()
	defer del()

	helper := exec.Command(os.Args[0], memifbridgeArg, dir)
	helperIn, e := helper.StdinPipe()
	require.NoError(e)
	helper.Stdout = os.Stdout
	helper.Stderr = os.Stderr
	require.NoError(helper.Start())
	time.Sleep(1 * time.Second)

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

	helperIn.Write([]byte("."))
	assert.NoError(helper.Wait())
}

const memifbridgeArg = "memifbridge"

func memifbridgeHelper() {
	dir := os.Args[2]
	locA := memiftransport.Locator{
		Role:       memiftransport.RoleServer,
		SocketName: path.Join(dir, "memifA.sock"),
		ID:         1216,
	}
	locB := memiftransport.Locator{
		Role:       memiftransport.RoleServer,
		SocketName: path.Join(dir, "memifB.sock"),
		ID:         2643,
	}

	bridge, e := memiftransport.NewBridge(locA, locB)
	if e != nil {
		panic(e)
	}

	io.ReadAtLeast(os.Stdin, make([]byte, 1), 1)
	bridge.Close()
}
