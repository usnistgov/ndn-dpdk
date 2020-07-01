package ifacetest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
)

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

	face2.D.Close()
	face1.D.Close()
	if assert.Len(faceClosedEvts, 2) {
		assert.Equal(id2, faceClosedEvts[0])
		assert.Equal(id1, faceClosedEvts[1])
	}
}
