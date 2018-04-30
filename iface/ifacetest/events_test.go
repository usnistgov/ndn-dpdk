package ifacetest

import (
	"testing"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/mockface"
)

func TestEvents(t *testing.T) {
	assert, _ := makeAR(t)

	var faceNewEvts []iface.FaceId
	var faceUpEvts []iface.FaceId
	var faceDownEvts []iface.FaceId
	var faceClosedEvts []iface.FaceId
	defer iface.OnFaceNew(func(id iface.FaceId) {
		faceNewEvts = append(faceNewEvts, id)
	}).Close()
	defer iface.OnFaceUp(func(id iface.FaceId) {
		faceUpEvts = append(faceUpEvts, id)
	}).Close()
	defer iface.OnFaceDown(func(id iface.FaceId) {
		faceDownEvts = append(faceDownEvts, id)
	}).Close()
	defer iface.OnFaceClosed(func(id iface.FaceId) {
		faceClosedEvts = append(faceClosedEvts, id)
	}).Close()

	face1 := mockface.New()
	face2 := mockface.New()
	id1, id2 := face1.GetFaceId(), face2.GetFaceId()
	if assert.Len(faceNewEvts, 2) {
		assert.Equal(id1, faceNewEvts[0])
		assert.Equal(id2, faceNewEvts[1])
	}

	face1.SetDown(true)
	face1.SetDown(true)
	if assert.Len(faceDownEvts, 1) {
		assert.Equal(id1, faceDownEvts[0])
	}
	face1.SetDown(false)
	face1.SetDown(false)
	if assert.Len(faceUpEvts, 1) {
		assert.Equal(id1, faceUpEvts[0])
	}

	face2.Close()
	face1.Close()
	if assert.Len(faceClosedEvts, 2) {
		assert.Equal(id2, faceClosedEvts[0])
		assert.Equal(id1, faceClosedEvts[1])
	}
}
