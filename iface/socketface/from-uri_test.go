package socketface_test

import (
	"testing"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/iface/socketface"
)

func TestMgmtCreateFace(t *testing.T) {
	assert, require := makeAR(t)

	rxg := socketface.NewRxGroup()
	txl := iface.NewTxLoop()
	createFace := socketface.MakeMgmtCreateFace(socketfaceCfg, rxg, txl, 64)

	id1, e := createFace(faceuri.MustParse("udp4://127.0.0.1:7001"), nil)
	require.NoError(e)
	assert.Equal([]iface.FaceId{id1}, rxg.ListFacesInRxLoop())

	face1 := iface.Get(id1)
	require.NotNil(face1)
	face1.Close()
	assert.Empty(rxg.ListFacesInRxLoop())

	// don't rxg.Close() or txl.StopTxLoop() because they are not running
}
