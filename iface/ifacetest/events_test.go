package ifacetest

import (
	"testing"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/mockface"
)

func TestEvents(t *testing.T) {
	assert, _ := makeAR(t)

	var faceNewEvts []iface.IFace
	var faceClosedEvts []iface.FaceId
	iface.OnFaceNew(func(face iface.IFace) {
		faceNewEvts = append(faceNewEvts, face)
	})
	iface.OnFaceClosed(func(faceId iface.FaceId) {
		faceClosedEvts = append(faceClosedEvts, faceId)
	})

	face1 := mockface.New()
	if assert.Len(faceNewEvts, 1) {
		assert.Equal(face1, faceNewEvts[0])
	}
	face2 := mockface.New()
	if assert.Len(faceNewEvts, 2) {
		assert.Equal(face2, faceNewEvts[1])
	}
	faceId1, faceId2 := face1.GetFaceId(), face2.GetFaceId()

	face2.Close()
	if assert.Len(faceClosedEvts, 1) {
		assert.Equal(faceId2, faceClosedEvts[0])
	}
	face1.Close()
	if assert.Len(faceClosedEvts, 2) {
		assert.Equal(faceId1, faceClosedEvts[1])
	}
}
