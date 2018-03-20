package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/app/fwdp/fwdptestfixture"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestInterestData(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()

	fixture.SetFibEntry("/B", face2.GetFaceId())
	fixture.SetFibEntry("/C", face3.GetFaceId())

	interestB1 := ndntestutil.MakeInterest("/B/1")
	face1.Rx(interestB1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 0)

	dataB1 := ndntestutil.MakeData("/B/1")
	ndntestutil.CopyPitToken(dataB1, face2.TxInterests[0])
	face2.Rx(dataB1)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 1)
	assert.Len(face1.TxNacks, 0)
}
