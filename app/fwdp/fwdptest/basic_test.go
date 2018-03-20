package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/app/fwdp/fwdptestfixture"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestInterestData(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()

	fixture.SetFibEntry("/B", face2.GetFaceId())
	fixture.SetFibEntry("/C", face3.GetFaceId())

	interestB := ndntestutil.MakeInterest("/B/2")
	face1.Rx(interestB)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 0)

	face2.Rx(ndntestutil.MakeData("/B/2"))
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 0) // cannot deliver due to missing PIT token
	assert.Len(face1.TxNacks, 0)
}
