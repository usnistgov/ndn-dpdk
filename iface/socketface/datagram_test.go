package socketface_test

import (
	"fmt"
	"net"
	"os"
	"testing"

	"golang.org/x/sys/unix"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/socketface"
)

func TestDatagram(t *testing.T) {
	assert, require := makeAR(t)

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
	assert.True(faceA.IsDatagram())

	fixture := ifacetestfixture.New(t, faceA, socketface.NewRxGroup(faceA), faceB)
	fixture.RunTest()
	fixture.CheckCounters()
}

func TestUdp(t *testing.T) {
	assert, require := makeAR(t)

	remoteUri := faceuri.MustParse("udp4://127.0.0.1:7000")
	face, e := socketface.NewFromUri(remoteUri, nil, socketface.Config{
		Mempools:    faceMempools,
		RxMp:        directMp,
		RxqCapacity: 64,
		TxqCapacity: 64,
	})
	require.NoError(e)
	defer face.Close()
	assert.True(face.IsDatagram())

	assert.Equal(iface.FaceKind_Socket, face.GetFaceId().GetKind())
	assert.Equal(fmt.Sprintf("udp4://%s", face.GetConn().LocalAddr()), face.GetLocalUri().String())
	assert.Equal("udp4://127.0.0.1:7000", face.GetRemoteUri().String())
}
