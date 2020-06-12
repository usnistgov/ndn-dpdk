package socketface_test

import (
	"testing"

	"golang.org/x/sys/unix"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ifacetestenv"
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

	fixture := ifacetestenv.New(t, faceA, faceB)
	fixture.RunTest()
	fixture.CheckCounters()
}

func TestUdp(t *testing.T) {
	assert, require := makeAR(t)

	loc := iface.MustParseLocator(`{ "Scheme": "udp", "remote": "127.0.0.1:7000" }`).(socketface.Locator)
	face, e := socketface.Create(loc, socketfaceCfg)
	require.NoError(e)
	defer face.Close()

	assert.Equal(iface.FaceKind_Socket, face.GetFaceId().GetKind())
	loc = face.GetLocator().(socketface.Locator)
	assert.Equal("udp", loc.Scheme)
	assert.Equal(face.GetConn().LocalAddr().String(), loc.Local)
	assert.Equal("127.0.0.1:7000", loc.Remote)
	ifacetestenv.CheckLocatorMarshal(t, loc)
}
