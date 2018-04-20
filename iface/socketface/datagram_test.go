package socketface_test

import (
	"fmt"
	"net"
	"os"
	"testing"

	"golang.org/x/sys/unix"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/socketface"
)

func TestDatagram(t *testing.T) {
	_, require := makeAR(t)

	fd, e := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	require.NoError(e)

	makeConnFromFd := func(fd int) net.Conn {
		file := os.NewFile(uintptr(fd), "")
		require.NotNil(file)
		defer file.Close()
		conn, e := net.FilePacketConn(file)
		require.NoError(e)
		return conn.(*net.UnixConn)
	}

	faceA := socketface.New(makeConnFromFd(fd[0]), socketface.Config{
		Mempools:    faceMempools,
		RxMp:        directMp,
		RxqCapacity: 64,
		TxqCapacity: 64,
	})
	faceB := socketface.New(makeConnFromFd(fd[1]), socketface.Config{
		Mempools:    faceMempools,
		RxMp:        directMp,
		RxqCapacity: 64,
		TxqCapacity: 64,
	})
	defer faceA.Close()
	defer faceB.Close()

	fixture := ifacetestfixture.New(t, faceA, socketface.NewRxGroup(faceA), faceB)
	fixture.RunTest()
	fixture.CheckCounters()
}

func TestUdp(t *testing.T) {
	assert, require := makeAR(t)

	addr, e := net.ResolveUDPAddr("udp", "127.0.0.1:7000")
	require.NoError(e)
	conn, e := net.DialUDP("udp", nil, addr)
	require.NoError(e)

	face := socketface.New(conn, socketface.Config{
		Mempools:    faceMempools,
		RxMp:        directMp,
		RxqCapacity: 64,
		TxqCapacity: 64,
	})
	defer face.Close()

	assert.Equal(iface.FaceKind_Socket, face.GetFaceId().GetKind())
	assert.Equal(fmt.Sprintf("udp4://%s", conn.LocalAddr()), face.GetLocalUri().String())
	assert.Equal("udp4://127.0.0.1:7000", face.GetRemoteUri().String())
}
