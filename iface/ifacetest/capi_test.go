package ifacetest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestCApiNoFace(t *testing.T) {
	assert, _ := makeAR(t)

	id := iface.AllocID() // non-existent face
	assert.True(Face_IsDown(id))

	pkts := make([]*ndni.Packet, 1)
	pkts[0] = ndnitestenv.MakeInterest("/A").AsPacket()
	iface.TxBurst(id, pkts) // should not crash
}

func TestCApi(t *testing.T) {
	assert, _ := makeAR(t)

	face := intface.MustNew()
	collect := intface.Collect(face)
	id := face.ID
	assert.False(Face_IsDown(id))

	face.SetDown(true)
	assert.True(Face_IsDown(id))
	face.SetDown(false)
	assert.False(Face_IsDown(id))

	pkts := make([]*ndni.Packet, 1)
	pkts[0] = ndnitestenv.MakeInterest("/A").AsPacket()
	iface.TxBurst(id, pkts)

	time.Sleep(100 * time.Millisecond)
	assert.Equal(1, collect.Count())

	face.D.Close()
}
