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
	ndntestutil.SetPitToken(interestB1, 0x0290dd7089e9d790)
	face1.Rx(interestB1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 0)

	dataB1 := ndntestutil.MakeData("/B/1")
	ndntestutil.CopyPitToken(dataB1, face2.TxInterests[0])
	face2.Rx(dataB1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face1.TxData, 1)
	assert.Len(face1.TxNacks, 0)
	assert.EqualValues(0x0290dd7089e9d790, ndntestutil.GetPitToken(face1.TxData[0]))
}

func TestInterestNack(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()

	interestA1 := ndntestutil.MakeInterest("/A/1")
	ndntestutil.SetPitToken(interestA1, 0x431328d8b4075167)
	face1.Rx(interestA1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face1.TxNacks, 1)
	assert.EqualValues(0x431328d8b4075167, ndntestutil.GetPitToken(face1.TxNacks[0]))
}
