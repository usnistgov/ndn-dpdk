package fwdptest

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestDataWrongName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interest := makeInterest("/B/1")
	face1.Rx(interest)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)

	data := makeData("/B/2", time.Second) // name does not match
	copyPitToken(data, face2.TxInterests[0])
	face2.Rx(data)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NDataMiss
	}))
}

func TestDataLongerName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interest := makeInterest("/B/1") // no CanBePrefix
	face1.Rx(interest)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)

	data := makeData("/B/1/Z", time.Second) // name has suffix
	copyPitToken(data, face2.TxInterests[0])
	face2.Rx(data)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NDataMiss
	}))
}

func TestDataZeroFreshnessPeriod(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interest := makeInterest("/B/1", ndn.MustBeFreshFlag) // has MustBeFresh
	face1.Rx(interest)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)

	data := makeData("/B/1") // no FreshnessPeriod
	copyPitToken(data, face2.TxInterests[0])
	face2.Rx(data)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NDataMiss
	}))
}

func TestNackWrongName(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interest := makeInterest("/B/1", uint32(0xdb22330b))
	face1.Rx(interest)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)

	nack := ndn.MakeNackFromInterest(makeInterest("/B/2", uint32(0xdb22330b)), ndn.NackReason_NoRoute)
	copyPitToken(nack, face2.TxInterests[0])
	face2.Rx(nack)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.GetFwdPit(i).ReadCounters().NNackMiss
	}))
}

func TestNackWrongNonce(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interest := makeInterest("/B/1", uint32(0x19c3e8b8))
	face1.Rx(interest)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)

	nack := ndn.MakeNackFromInterest(makeInterest("/B/1", uint32(0xf4d9aad1)), ndn.NackReason_NoRoute)
	copyPitToken(nack, face2.TxInterests[0])
	face2.Rx(nack)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxData, 0)
	assert.Len(face1.TxNacks, 0)

	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.ReadFwdInfo(i).NNackMismatch
	}))
}
