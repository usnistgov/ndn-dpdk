package iface_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"github.com/usnistgov/ndn-dpdk/ndni/ndnitestenv"
)

func TestTxBurst(t *testing.T) {
	assert, _ := makeAR(t)

	face := intface.MustNew()
	collect := intface.Collect(face)
	id := face.ID

	face.SetDown(true)
	pkts := make([]*ndni.Packet, 1)
	pkts[0] = ndnitestenv.MakeInterest("/A").AsPacket()
	iface.TxBurst(id, pkts)

	face.SetDown(false)
	pkts = make([]*ndni.Packet, 1)
	pkts[0] = ndnitestenv.MakeData("/A").AsPacket()
	iface.TxBurst(id, pkts)
	time.Sleep(100 * time.Millisecond)

	face.D.Close()
	pkts = make([]*ndni.Packet, 1)
	pkts[0] = ndnitestenv.MakeInterest("/A").AsPacket()
	iface.TxBurst(id, pkts)

	assert.Equal(1, collect.Count())
	assert.NotNil(collect.Get(0).Data)
}

func TestEvents(t *testing.T) {
	assert, _ := makeAR(t)

	var faceNewEvts []iface.ID
	var faceUpEvts []iface.ID
	var faceDownEvts []iface.ID
	var faceClosingEvts []iface.ID
	var faceClosedEvts []iface.ID
	defer iface.OnFaceNew(func(id iface.ID) {
		faceNewEvts = append(faceNewEvts, id)
	}).Close()
	defer iface.OnFaceUp(func(id iface.ID) {
		faceUpEvts = append(faceUpEvts, id)
	}).Close()
	defer iface.OnFaceDown(func(id iface.ID) {
		faceDownEvts = append(faceDownEvts, id)
	}).Close()
	defer iface.OnFaceClosing(func(id iface.ID) {
		faceClosingEvts = append(faceClosingEvts, id)
	}).Close()
	defer iface.OnFaceClosed(func(id iface.ID) {
		faceClosedEvts = append(faceClosedEvts, id)
		assert.Len(faceClosingEvts, len(faceClosedEvts))
		assert.Equal(id, faceClosedEvts[len(faceClosingEvts)-1])
	}).Close()

	face1 := intface.MustNew()
	face2 := intface.MustNew()
	id1, id2 := face1.ID, face2.ID
	if assert.Len(faceNewEvts, 2) {
		assert.Equal(id1, faceNewEvts[0])
		assert.Equal(id2, faceNewEvts[1])
	}

	assert.False(iface.IsDown(id1))
	face1.SetDown(true)
	assert.True(iface.IsDown(id1))
	face1.SetDown(true)
	if assert.Len(faceDownEvts, 1) {
		assert.Equal(id1, faceDownEvts[0])
	}

	face1.SetDown(false)
	assert.False(iface.IsDown(id1))
	face1.SetDown(false)
	if assert.Len(faceUpEvts, 1) {
		assert.Equal(id1, faceUpEvts[0])
	}

	face2.D.Close()
	face1.D.Close()
	if assert.Len(faceClosedEvts, 2) {
		assert.Equal(id2, faceClosedEvts[0])
		assert.Equal(id1, faceClosedEvts[1])
	}
	assert.True(iface.IsDown(id1))
}
