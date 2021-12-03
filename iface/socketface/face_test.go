package socketface_test

import (
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/gabstv/freeport"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/testenv"
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

	portA, portB := 0, 0
	for portA == portB {
		portA, _ = freeport.UDP()
		portB, _ = freeport.UDP()
	}
	addrA := "127.0.0.1:" + strconv.Itoa(portA)
	addrB := "127.0.0.1:" + strconv.Itoa(portB)

	locA := mustParseLocator(`{ "scheme": "udp", "local": "` + addrA + `", "remote": "` + addrB + `" }`)
	ifacetestenv.CheckLocatorMarshal(t, locA)
	faceA, e := socketface.New(locA)
	require.NoError(e)
	defer faceA.Close()

	locB := mustParseLocator(`{ "scheme": "udp", "local": "` + addrB + `", "remote": "` + addrA + `" }`)
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

func checkStreamRedialing(t *testing.T, listener net.Listener, makeFaceA func() iface.Face) {
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

	var cntMap map[string]interface{}
	require.NoError(jsonhelper.Roundtrip(faceA.ReadExCounters(), &cntMap))
	assert.InDelta(1.5, cntMap["nRedials"], 0.6) // redial counter should be 1 or 2
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
		loc := mustParseLocator(fmt.Sprintf(`{ "scheme": "tcp", "remote": "127.0.0.1:%d" }`, addr.Port))
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

	addr, del := testenv.TempName("unix.sock")
	defer del()
	listener, e := net.Listen("unix", addr)
	require.NoError(e)
	defer listener.Close()

	checkStreamRedialing(t, listener, func() iface.Face {
		loc := mustParseLocator(fmt.Sprintf(`{ "scheme": "unix", "remote": "%s" }`, addr))
		face, e := socketface.New(loc)
		require.NoError(e)

		loc = face.Locator().(socketface.Locator)
		assert.Equal("unix", loc.Scheme())
		assert.Equal(addr, loc.Remote)
		ifacetestenv.CheckLocatorMarshal(t, loc)

		return face
	})
}
