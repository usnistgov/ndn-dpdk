package sockettransport_test

import (
	"net"
	"sync"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/sockettransport"
)

var trCfg = sockettransport.Config{
	RxQueueSize: 64,
	TxQueueSize: 64,
}

func TestUdp(t *testing.T) {
	_, require := makeAR(t)

	var dialer sockettransport.Dialer
	dialer.Config = trCfg

	trA, e := dialer.Dial("udp", "127.0.0.1:7001", "127.0.0.1:7002")
	require.NoError(e)
	trB, e := dialer.Dial("udp", "127.0.0.1:7002", "127.0.0.1:7001")
	require.NoError(e)

	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}

func TestTcp(t *testing.T) {
	_, require := makeAR(t)

	listener, e := net.Listen("tcp", "127.0.0.1:7002")
	require.NoError(e)
	defer listener.Close()

	var trA, trB *sockettransport.Transport

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		var dialer sockettransport.Dialer
		dialer.Config = trCfg
		tr, e := dialer.Dial("tcp", "", "127.0.0.1:7002")
		require.NoError(e)
		trA = tr
		wg.Done()
	}()

	go func() {
		socket, e := listener.Accept()
		require.NoError(e)
		tr, e := sockettransport.New(socket, trCfg)
		require.NoError(e)
		trB = tr
		wg.Done()
	}()

	wg.Wait()
	var c ndntestenv.L3FaceTester
	c.CheckTransport(t, trA, trB)
}
