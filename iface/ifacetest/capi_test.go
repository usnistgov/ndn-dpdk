package ifacetest

import (
	"testing"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestCApiNoFace(t *testing.T) {
	assert, _ := makeAR(t)

	id := iface.AllocId(iface.FaceKind_Mock) // non-existent face
	assert.True(Face_IsDown(id))

	pkts := make([]ndn.Packet, 1)
	pkts[0] = ndntestutil.MakeInterest("/A").GetPacket()
	Face_TxBurst(id, pkts) // should not crash
}

func TestCApi(t *testing.T) {
	assert, _ := makeAR(t)

	face := mockface.New()
	defer face.Close()
	id := face.GetFaceId()
	assert.False(Face_IsDown(id))

	face.SetDown(true)
	assert.True(Face_IsDown(id))
	face.SetDown(false)
	assert.False(Face_IsDown(id))

	pkts := make([]ndn.Packet, 1)
	pkts[0] = ndntestutil.MakeInterest("/A").GetPacket()
	Face_TxBurst(id, pkts)

	assert.Len(face.TxInterests, 1)
}
