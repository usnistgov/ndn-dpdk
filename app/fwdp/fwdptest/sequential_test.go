package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/ndn/ndntestutil"
)

func TestSequential(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	face4 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "sequential", face1.GetFaceId(), face2.GetFaceId(), face3.GetFaceId())

	interest1 := ndntestutil.MakeInterest("/A/1")
	face4.Rx(interest1)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 0)
	assert.Len(face3.TxInterests, 0)

	interest2 := ndntestutil.MakeInterest("/A/1")
	face4.Rx(interest2)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 0)

	interest3 := ndntestutil.MakeInterest("/A/1")
	face4.Rx(interest3)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 1)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 1)

	interest4 := ndntestutil.MakeInterest("/A/1")
	face4.Rx(interest4)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 2)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 1)

	face2.SetDown(true)

	interest5 := ndntestutil.MakeInterest("/A/1")
	face4.Rx(interest5)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 2)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 2)
}
