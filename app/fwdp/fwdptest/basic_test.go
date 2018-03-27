package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/app/fwdp/fwdptestfixture"
	"ndn-dpdk/ndn"
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
	assert.Equal(uint64(0x0290dd7089e9d790), ndntestutil.GetPitToken(face1.TxData[0]))
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
	assert.Equal(uint64(0x431328d8b4075167), ndntestutil.GetPitToken(face1.TxNacks[0]))
}

func TestInterestCsHit(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", face2.GetFaceId())

	interestB1 := ndntestutil.MakeInterest("/B/1")
	ndntestutil.SetPitToken(interestB1, 0x193d673cdb9f85ac)
	face1.Rx(interestB1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)

	dataB1 := ndntestutil.MakeData("/B/1")
	ndntestutil.CopyPitToken(dataB1, face2.TxInterests[0])
	face2.Rx(dataB1)
	time.Sleep(100 * time.Millisecond)
	require.Len(face1.TxData, 1)
	assert.Equal(uint64(0x193d673cdb9f85ac), ndntestutil.GetPitToken(face1.TxData[0]))
	assert.Equal(0*time.Millisecond, face1.TxData[0].GetFreshnessPeriod())

	interestB1mbf := ndntestutil.MakeInterest("/B/1", ndn.MustBeFreshFlag)
	ndntestutil.SetPitToken(interestB1mbf, 0xf716737325e04a77)
	face1.Rx(interestB1mbf)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 2)

	dataB1fp := ndntestutil.MakeData("/B/1", 2500*time.Millisecond)
	ndntestutil.CopyPitToken(dataB1fp, face2.TxInterests[1])
	face2.Rx(dataB1fp)
	time.Sleep(100 * time.Millisecond)
	require.Len(face1.TxData, 2)
	assert.Equal(uint64(0xf716737325e04a77), ndntestutil.GetPitToken(face1.TxData[1]))
	assert.Equal(2500*time.Millisecond, face1.TxData[1].GetFreshnessPeriod())

	interestB1 = ndntestutil.MakeInterest("/B/1")
	ndntestutil.SetPitToken(interestB1, 0xaec62dad2f669e6b)
	face1.Rx(interestB1)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face2.TxInterests, 2)
	require.Len(face1.TxData, 3)
	assert.Equal(uint64(0xaec62dad2f669e6b), ndntestutil.GetPitToken(face1.TxData[2]))
	assert.Equal(2500*time.Millisecond, face1.TxData[2].GetFreshnessPeriod())

	interestB1mbf = ndntestutil.MakeInterest("/B/1")
	ndntestutil.SetPitToken(interestB1, 0xb5565a4e715c858d)
	face1.Rx(interestB1)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face2.TxInterests, 2)
	require.Len(face1.TxData, 4)
	assert.Equal(uint64(0xb5565a4e715c858d), ndntestutil.GetPitToken(face1.TxData[3]))
	assert.Equal(2500*time.Millisecond, face1.TxData[3].GetFreshnessPeriod())
}
