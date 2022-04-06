package socketface_test

import (
	"encoding/json"
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ifacetestenv"
	"github.com/usnistgov/ndn-dpdk/iface/socketface"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
	"go4.org/must"
)

func mustParseLocator(input string) socketface.Locator {
	var locw iface.LocatorWrapper
	if e := json.Unmarshal([]byte(input), &locw); e != nil {
		panic(e)
	}
	return locw.Locator.(socketface.Locator)
}

func TestUDP(t *testing.T) {
	assert, require := makeAR(t)
	fixture := ifacetestenv.NewFixture(t)

	var addrA, addrB string
	{
		addr, e := net.ResolveUDPAddr("udp", "127.0.0.1:0")
		require.NoError(e)
		listenerA, e := net.ListenUDP("udp", addr)
		require.NoError(e)
		listenerB, e := net.ListenUDP("udp", addr)
		require.NoError(e)
		addrA, addrB = listenerA.LocalAddr().String(), listenerB.LocalAddr().String()
		listenerA.Close()
		listenerB.Close()
	}

	locA := mustParseLocator(`{"scheme":"udp", "local":"` + addrA + `", "remote":"` + addrB + `"}`)
	ifacetestenv.CheckLocatorMarshal(t, locA)
	faceA, e := socketface.New(locA)
	require.NoError(e)
	defer faceA.Close()

	locB := mustParseLocator(`{"scheme":"udp", "local":"` + addrB + `", "remote":"` + addrA + `"}`)
	faceB, e := socketface.New(locB)
	require.NoError(e)
	defer faceB.Close()

	locA = faceA.Locator().(socketface.Locator)
	assert.Equal("udp", locA.Scheme())
	assert.Equal(addrA, locA.Local)
	assert.Equal(addrB, locA.Remote)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()
}

func checkStreamRedialing(t testing.TB, listener net.Listener, makeFaceA func() iface.Face) {
	assert, require := makeAR(t)
	fixture := ifacetestenv.NewFixture(t)

	faceA := makeFaceA()
	defer faceA.Close()

	var hasDownEvt, hasUpEvt bool
	defer iface.OnFaceDown(func(id iface.ID) {
		if id == faceA.ID() {
			hasDownEvt = true
		}
	})()
	defer iface.OnFaceUp(func(id iface.ID) {
		if id == faceA.ID() {
			hasUpEvt = true
		}
	})()

	accepted, e := listener.Accept()
	require.NoError(e)

	innerB, e := sockettransport.New(accepted, sockettransport.Config{})
	require.NoError(e)
	faceB, e := socketface.Wrap(innerB, socketface.Config{})
	require.NoError(e)

	fixture.RunTest(faceA, faceB)
	fixture.CheckCounters()

	must.Close(accepted)            // close initial connection
	accepted, e = listener.Accept() // faceA should redial
	require.NoError(e)
	time.Sleep(100 * time.Millisecond)
	must.Close(accepted)

	assert.True(hasDownEvt)
	assert.True(hasUpEvt)

	var cnt sockettransport.Counters
	require.NoError(jsonhelper.Roundtrip(faceA.ExCounters(), &cnt))
	assert.InDelta(1.5, float64(cnt.NRedials), 0.6) // redial counter should be 1 or 2
}

func TestTCP(t *testing.T) {
	assert, require := makeAR(t)

	addr, e := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	require.NoError(e)
	listener, e := net.ListenTCP("tcp", addr)
	require.NoError(e)
	defer listener.Close()
	*addr = *listener.Addr().(*net.TCPAddr)

	checkStreamRedialing(t, listener, func() iface.Face {
		loc := mustParseLocator(fmt.Sprintf(`{"scheme":"tcp", "remote":"127.0.0.1:%d"}`, addr.Port))
		face, e := socketface.New(loc)
		require.NoError(e)

		loc = face.Locator().(socketface.Locator)
		assert.Equal("tcp", loc.Scheme())
		assert.Equal(fmt.Sprintf("127.0.0.1:%d", addr.Port), loc.Remote)
		ifacetestenv.CheckLocatorMarshal(t, loc)

		return face
	})
}

func TestUnix(t *testing.T) {
	assert, require := makeAR(t)
	addr := filepath.Join(t.TempDir(), "unix.sock")

	listener, e := net.Listen("unix", addr)
	require.NoError(e)
	defer listener.Close()

	checkStreamRedialing(t, listener, func() iface.Face {
		loc := mustParseLocator(fmt.Sprintf(`{"scheme":"unix", "remote":"%s"}`, addr))
		face, e := socketface.New(loc)
		require.NoError(e)

		loc = face.Locator().(socketface.Locator)
		assert.Equal("unix", loc.Scheme())
		assert.Equal(addr, loc.Remote)
		ifacetestenv.CheckLocatorMarshal(t, loc)

		return face
	})
}
