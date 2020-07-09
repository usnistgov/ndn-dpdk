package socketface_test

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

func TestUdp(t *testing.T) {
	assert, require := makeAR(t)
	fixture := ifacetestenv.New(t)
	defer fixture.Close()

	locA := iface.MustParseLocator(`{ "Scheme": "udp", "Local": "127.0.0.1:7001", "Remote": "127.0.0.1:7002" }`).(socketface.Locator)
	ifacetestenv.CheckLocatorMarshal(t, locA)
	faceA, e := socketface.New(locA, socketfaceCfg)
	require.NoError(e)
	defer faceA.Close()

	locB := iface.MustParseLocator(`{ "Scheme": "udp", "Local": "127.0.0.1:7002", "Remote": "127.0.0.1:7001" }`).(socketface.Locator)
	faceB, e := socketface.New(locB, socketfaceCfg)
	require.NoError(e)
	defer faceB.Close()

	locA = faceA.Locator().(socketface.Locator)
	assert.Equal("udp", locA.Scheme)
	assert.Equal("127.0.0.1:7001", locA.Local)
	assert.Equal("127.0.0.1:7002", locA.Remote)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()
}

func checkStreamRedialing(t *testing.T, listener net.Listener, makeFaceA func() iface.Face) {
	assert, require := makeAR(t)
	fixture := ifacetestenv.New(t)
	defer fixture.Close()

	faceA := makeFaceA()
	defer faceA.Close()

	var hasDownEvt, hasUpEvt bool
	defer iface.OnFaceDown(func(id iface.ID) {
		if id == faceA.ID() {
			hasDownEvt = true
		}
	}).Close()
	defer iface.OnFaceUp(func(id iface.ID) {
		if id == faceA.ID() {
			hasUpEvt = true
		}
	}).Close()

	accepted, e := listener.Accept()
	require.NoError(e)

	innerB, e := sockettransport.New(accepted, sockettransport.Config{})
	require.NoError(e)
	faceB, e := socketface.Wrap(innerB, socketfaceCfg)
	require.NoError(e)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()

	accepted.Close()                // close initial connection
	accepted, e = listener.Accept() // faceA should redial
	require.NoError(e)
	time.Sleep(100 * time.Millisecond)
	accepted.Close()

	assert.True(hasDownEvt)
	assert.True(hasUpEvt)

	cnt := faceA.ReadExCounters().(socketface.ExCounters)
	assert.InDelta(1.5, float64(cnt.NRedials), 0.6) // redial counter should be 1 or 2
}

func TestTcp(t *testing.T) {
	assert, require := makeAR(t)

	addr, e := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	require.NoError(e)
	listener, e := net.ListenTCP("tcp", addr)
	require.NoError(e)
	defer listener.Close()
	*addr = *listener.Addr().(*net.TCPAddr)

	checkStreamRedialing(t, listener, func() iface.Face {
		loc := iface.MustParseLocator(fmt.Sprintf(`{ "Scheme": "tcp", "Remote": "127.0.0.1:%d" }`, addr.Port)).(socketface.Locator)
		face, e := socketface.New(loc, socketfaceCfg)
		require.NoError(e)

		loc = face.Locator().(socketface.Locator)
		assert.Equal("tcp", loc.Scheme)
		assert.Equal(fmt.Sprintf("127.0.0.1:%d", addr.Port), loc.Remote)
		ifacetestenv.CheckLocatorMarshal(t, loc)

		return face
	})
}

func TestUnix(t *testing.T) {
	assert, require := makeAR(t)

	tmpdir, e := ioutil.TempDir("", "socketface-test")
	require.NoError(e)
	defer os.RemoveAll(tmpdir)
	addr := path.Join(tmpdir, "unix.sock")
	listener, e := net.Listen("unix", addr)
	require.NoError(e)
	defer listener.Close()

	checkStreamRedialing(t, listener, func() iface.Face {
		loc := iface.MustParseLocator(fmt.Sprintf(`{ "Scheme": "unix", "Remote": "%s" }`, addr)).(socketface.Locator)
		face, e := socketface.New(loc, socketfaceCfg)
		require.NoError(e)

		loc = face.Locator().(socketface.Locator)
		assert.Equal("unix", loc.Scheme)
		assert.Equal(addr, loc.Remote)
		ifacetestenv.CheckLocatorMarshal(t, loc)

		return face
	})
}
