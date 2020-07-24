package ethface_test

import (
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/iface/ethface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

func TestMemif(t *testing.T) {
	assert, require := makeAR(t)

	dir, e := ioutil.TempDir("", "ethface-test")
	require.NoError(e)
	defer os.RemoveAll(dir)
	socketName := path.Join(dir, "memif.sock")

	fixture := ifacetestenv.NewFixture(t)
	defer fixture.Close()

	var locA ethface.Locator
	locA.Local = memiftransport.AddressDPDK
	locA.Remote = memiftransport.AddressApp
	locA.Memif = &memiftransport.Locator{
		SocketName: socketName,
		ID:         0,
	}
	faceA, e := locA.CreateFace()
	require.NoError(e)
	assert.Equal("memif", faceA.Locator().Scheme())

	var locB ethface.Locator
	locB.Local = memiftransport.AddressDPDK
	locB.Remote = memiftransport.AddressApp
	locB.Memif = &memiftransport.Locator{
		SocketName: socketName,
		ID:         1,
	}
	faceB, e := locB.CreateFace()
	require.NoError(e)

	helper := exec.Command(os.Args[0], memifbridgeArg, socketName)
	helperIn, e := helper.StdinPipe()
	require.NoError(e)
	helper.Stdout = os.Stdout
	helper.Stderr = os.Stderr
	require.NoError(helper.Start())
	time.Sleep(5 * time.Second)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()

	helperIn.Write([]byte("."))
	assert.NoError(helper.Wait())
}

const memifbridgeArg = "memifbridge"

func memifbridgeHelper(socketName string) {
	var locA, locB memiftransport.Locator
	locA.SocketName = socketName
	locA.ID = 0
	locB = locA
	locB.ID = 1

	trA, e := memiftransport.New(locA)
	if e != nil {
		os.Exit(3)
	}

	trB, e := memiftransport.New(locB)
	if e != nil {
		os.Exit(4)
	}

	go func() {
		for pkt := range trA.Rx() {
			trB.Tx() <- pkt
		}
		close(trB.Tx())
	}()

	go func() {
		for pkt := range trB.Rx() {
			trA.Tx() <- pkt
		}
		close(trA.Tx())
	}()

	io.ReadAtLeast(os.Stdin, make([]byte, 1), 1)
}
