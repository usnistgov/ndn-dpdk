package sockettransport_test

import (
	"net"
	"path/filepath"
	"sync"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

func TestPipe(t *testing.T) {
	_, require := makeAR(t)

	trA, trB, e := sockettransport.Pipe(sockettransport.Config{})
	require.NoError(e)

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}

func TestUDP(t *testing.T) {
	assert, require := makeAR(t)

	trA, e := sockettransport.Dial("udp", "127.0.0.1:7001", "127.0.0.1:7002", sockettransport.Config{})
	require.NoError(e)
	trB, e := sockettransport.Dial("udp", "127.0.0.1:7002", "127.0.0.1:7001", sockettransport.Config{})
	require.NoError(e)

	// REUSEADDR
	trC, e := sockettransport.Dial("udp", "127.0.0.1:7001", "127.0.0.1:7003", sockettransport.Config{})
	if assert.NoError(e) {
		trC.Close()
	}

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}

func TestTCP(t *testing.T) {
	_, require := makeAR(t)

	listener, e := net.Listen("tcp", "127.0.0.1:7002")
	require.NoError(e)
	defer listener.Close()

	checkStream(t, listener)
}

func TestUnix(t *testing.T) {
	_, require := makeAR(t)
	addr := filepath.Join(t.TempDir(), "unix.sock")

	listener, e := net.Listen("unix", addr)
	require.NoError(e)
	defer listener.Close()

	checkStream(t, listener)
}

func checkStream(t testing.TB, listener net.Listener) {
	_, require := makeAR(t)

	var trA, trB sockettransport.Transport
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		listenAddr := listener.Addr()
		tr, e := sockettransport.Dial(listenAddr.Network(), "", listenAddr.String(), sockettransport.Config{})
		require.NoError(e)
		trA = tr
	}()

	go func() {
		defer wg.Done()
		socket, e := listener.Accept()
		require.NoError(e)
		tr, e := sockettransport.New(socket, sockettransport.Config{})
		require.NoError(e)
		trB = tr
	}()

	wg.Wait()

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}
