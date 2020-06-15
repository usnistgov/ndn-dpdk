package ifacetest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/mockface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestCApiNoFace(t *testing.T) {
	assert, _ := makeAR(t)

	id := iface.AllocId(iface.FaceKind_Mock) // non-existent face
	assert.True(Face_IsDown(id))

	pkts := make([]*ndn.Packet, 1)
	pkts[0] = ndntestenv.MakeInterest("/A").GetPacket()
	Face_TxBurst(id, pkts) // should not crash
}

func TestCApi(t *testing.T) {
	assert, _ := makeAR(t)

	face := mockface.New()
	id := face.GetFaceId()
	assert.False(Face_IsDown(id))

	face.SetDown(true)
	assert.True(Face_IsDown(id))
	face.SetDown(false)
	assert.False(Face_IsDown(id))

	txl := iface.NewTxLoop(eal.NumaSocket{})
	txl.SetLCore(eal.ListSlaveLCores()[0])
	txl.Launch()
	time.Sleep(10 * time.Millisecond)
	txl.AddFace(face)
	time.Sleep(90 * time.Millisecond)

	pkts := make([]*ndn.Packet, 1)
	pkts[0] = ndntestenv.MakeInterest("/A").GetPacket()
	Face_TxBurst(id, pkts)

	time.Sleep(100 * time.Millisecond)
	assert.Len(face.TxInterests, 1)

	txl.Stop()
	txl.Close()
	face.Close()
}
