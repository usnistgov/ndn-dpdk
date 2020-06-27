package socketface_test

import (
	"net"
	"sync"
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
	"github.com/usnistgov/ndn-dpdk/ndn/socketface"
)

var socketfaceCfg = socketface.Config{
	RxQueueSize: 64,
	TxQueueSize: 64,
}

func TestUdp(t *testing.T) {
	_, require := makeAR(t)

	var dialer socketface.Dialer
	dialer.Config = socketfaceCfg

	faceA, e := dialer.Dial("udp", "127.0.0.1:7001", "127.0.0.1:7002")
	require.NoError(e)
	faceB, e := dialer.Dial("udp", "127.0.0.1:7002", "127.0.0.1:7001")
	require.NoError(e)

	var c ndntestenv.CheckL3Face
	c.Execute(t, faceA, faceB)
}

func TestTcp(t *testing.T) {
	_, require := makeAR(t)

	listener, e := net.Listen("tcp", "127.0.0.1:7002")
	require.NoError(e)
	defer listener.Close()

	var faceA, faceB *socketface.SocketFace

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		var dialer socketface.Dialer
		dialer.Config = socketfaceCfg
		face, e := dialer.Dial("tcp", "", "127.0.0.1:7002")
		require.NoError(e)
		faceA = face
		wg.Done()
	}()

	go func() {
		socket, e := listener.Accept()
		require.NoError(e)
		face, e := socketface.New(socket, socketfaceCfg)
		require.NoError(e)
		faceB = face
		wg.Done()
	}()

	wg.Wait()
	var c ndntestenv.CheckL3Face
	c.Execute(t, faceA, faceB)
}
