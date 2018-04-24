package socketface_test

import (
	"fmt"
	"testing"

	"golang.org/x/sys/unix"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/ifacetestfixture"
	"ndn-dpdk/iface/socketface"
)

func TestDatagram(t *testing.T) {
	_, require := makeAR(t)

	fd, e := unix.Socketpair(unix.AF_UNIX, unix.SOCK_DGRAM, 0)
	require.NoError(e)

	faceA, e := socketface.New(makeConnFromFd(fd[0]), socketfaceCfg)
	require.NoError(e)
	defer faceA.Close()
	faceB, e := socketface.New(makeConnFromFd(fd[1]), socketfaceCfg)
	require.NoError(e)
	defer faceB.Close()

	fixture := ifacetestfixture.New(t, faceA, socketface.NewRxGroup(faceA), faceB)
	fixture.RunTest()
	fixture.CheckCounters()
}

func TestUdp(t *testing.T) {
	assert, require := makeAR(t)

	remoteUri := faceuri.MustParse("udp4://127.0.0.1:7000")
	face, e := socketface.NewFromUri(remoteUri, nil, socketfaceCfg)
	require.NoError(e)
	defer face.Close()

	assert.Equal(iface.FaceKind_Socket, face.GetFaceId().GetKind())
	assert.Equal(fmt.Sprintf("udp4://%s", face.GetConn().LocalAddr()), face.GetLocalUri().String())
	assert.Equal("udp4://127.0.0.1:7000", face.GetRemoteUri().String())
}
