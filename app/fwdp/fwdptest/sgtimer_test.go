package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/ndn/ndntestutil"
)

func TestSgTimer(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "delay", face2.GetFaceId())

	interest1 := ndntestutil.MakeInterest("/A/1", uint32(0x3979d1f6))
	face1.Rx(interest1)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face2.TxInterests, 0)
	time.Sleep(150 * time.Millisecond)
	assert.Len(face2.TxInterests, 1)
}
