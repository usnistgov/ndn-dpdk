package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/ndn"
)

func TestFastroute(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	face4 := fixture.CreateFace()
	fixture.SetFibEntry("/A/B", "fastroute", face1.GetFaceId(), face2.GetFaceId(), face3.GetFaceId())

	// multicast first Interest
	interest1 := makeInterest("/A/B/1")
	face4.Rx(interest1)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 1)

	// face3 replies Data
	data1 := makeData("/A/B/1")
	copyPitToken(data1, face3.TxInterests[0])
	face3.Rx(data1)
	time.Sleep(STEP_DELAY)

	// unicast to face3
	interest2 := makeInterest("/A/B/2")
	face4.Rx(interest2)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 2)

	// unicast to face3
	interest3 := makeInterest("/A/B/3")
	face4.Rx(interest3)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 3)

	// face3 fails
	face3.SetDown(true)

	// multicast next Interest because face3 failed
	interest4 := makeInterest("/A/B/4")
	face4.Rx(interest4)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 2)
	assert.Len(face2.TxInterests, 2)
	assert.Len(face3.TxInterests, 3) // no Interest to face3 because it's DOWN

	// face1 replies Data
	data4 := makeData("/A/B/4")
	copyPitToken(data4, face1.TxInterests[1])
	face1.Rx(data4)
	time.Sleep(STEP_DELAY)

	// unicast to face1
	interest5 := makeInterest("/A/B/5", uint32(0x422e9f49))
	face4.Rx(interest5)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 3)
	assert.Len(face2.TxInterests, 2)
	assert.Len(face3.TxInterests, 3)

	// face1 replies Nack~NoRoute, retry on other faces
	nack5 := ndn.MakeNackFromInterest(makeInterest("/A/B/5", uint32(0x422e9f49)), ndn.NackReason_NoRoute)
	copyPitToken(nack5, face1.TxInterests[2])
	face1.Rx(nack5)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 3)
	assert.Len(face2.TxInterests, 3)
	assert.Len(face3.TxInterests, 3) // no Interest to face3 because it's DOWN

	// face2 replies Nack~NoRoute as well, return Nack to downstream
	nack5 = ndn.MakeNackFromInterest(makeInterest("/A/B/5", uint32(0x422e9f49)), ndn.NackReason_NoRoute)
	copyPitToken(nack5, face2.TxInterests[2])
	face2.Rx(nack5)
	time.Sleep(STEP_DELAY)
	assert.Len(face4.TxNacks, 1)

	// multicast next Interest because faces Nacked
	interest6 := makeInterest("/A/B/6")
	face4.Rx(interest6)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 4)
	assert.Len(face2.TxInterests, 4)
	assert.Len(face3.TxInterests, 3) // no Interest to face3 because it's DOWN
}
