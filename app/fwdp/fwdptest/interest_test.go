package fwdptest

import (
	"bytes"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestInterestData(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())
	fixture.SetFibEntry("/C", "multicast", face3.GetFaceId())

	interest := makeInterest("/B/1")
	setPitToken(interest, 0x0290dd7089e9d790)
	face1.Rx(interest)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 0)

	data := makeData("/B/1")
	copyPitToken(data, face2.TxInterests[0])
	face2.Rx(data)
	time.Sleep(STEP_DELAY)
	require.Len(face1.TxData, 1)
	assert.Len(face1.TxNacks, 0)
	assert.Equal(uint64(0x0290dd7089e9d790), getPitToken(face1.TxData[0]))

	fibCnt := fixture.ReadFibCounters("/B")
	assert.Equal(uint64(1), fibCnt.NRxInterests)
	assert.Equal(uint64(1), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(1), fibCnt.NTxInterests)
}

func TestInterestDupNonce(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "multicast", face3.GetFaceId())

	interest := makeInterest("/A/1", uint32(0x6f937a51))
	setPitToken(interest, 0x3bddf54cffbc6ad0)
	face1.Rx(interest)
	time.Sleep(STEP_DELAY)
	assert.Len(face3.TxInterests, 1)

	interest = makeInterest("/A/1", uint32(0x6f937a51))
	setPitToken(interest, 0x3bddf54cffbc6ad0)
	face2.Rx(interest)
	time.Sleep(STEP_DELAY)
	require.Len(face3.TxInterests, 1)
	require.Len(face2.TxNacks, 1)
	assert.Equal(ndn.NackReason_Duplicate, face2.TxNacks[0].GetReason())
	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.ReadFwdInfo(i).NDupNonce
	}))

	data := makeData("/A/1")
	copyPitToken(data, face3.TxInterests[0])
	face3.Rx(data)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxData, 1)
	assert.Len(face1.TxNacks, 0)
	assert.Len(face2.TxData, 0)
	assert.Len(face2.TxNacks, 1)
}

func TestInterestSuppress(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "multicast", face3.GetFaceId())

	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		for i := 0; i < 400; i++ {
			<-ticker.C
			interest := makeInterest("/A/1")
			setPitToken(interest, 0xf4aab9f23eb5271e^uint64(i))
			if i%2 == 0 {
				face1.Rx(interest)
			} else {
				face2.Rx(interest)
			}
		}
		ticker.Stop()
	}()

	time.Sleep(500 * time.Millisecond)
	assert.InDelta(7, len(face3.TxInterests), 1)
	// suppression config is min=10, multiplier=2, max=100,
	// so 7 Interests should be forwarded at 0, 10, 30, 70, 150, 250, 350,
	// but this could be off by one on a slower machine.
}

func TestInterestNoRoute(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()

	interestA1 := makeInterest("/A/1")
	setPitToken(interestA1, 0x431328d8b4075167)
	face1.Rx(interestA1)
	time.Sleep(STEP_DELAY)
	require.Len(face1.TxNacks, 1)
	assert.Equal(uint64(0x431328d8b4075167), getPitToken(face1.TxNacks[0]))
	assert.Equal(ndn.NackReason_NoRoute, face1.TxNacks[0].GetReason())
	assert.Equal(uint64(1), fixture.SumCounter(func(dp *fwdp.DataPlane, i int) uint64 {
		return dp.ReadFwdInfo(i).NNoFibMatch
	}))
}

func TestHopLimit(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	face4 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "multicast", face1.GetFaceId())

	// cannot test HopLimit=0 because it's rejected by decoder,
	// so MakeInterest would fail

	// HopLimit becomes zero, cannot forward
	interest1 := makeInterest("/A/1", uint8(1))
	face2.Rx(interest1)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 0)

	// HopLimit is 1 after decrementing, forwarded with HopLimit=1
	interest2 := makeInterest("/A/1", uint8(2))
	face3.Rx(interest2)
	time.Sleep(STEP_DELAY)
	require.Len(face1.TxInterests, 1)
	assert.Equal(uint8(1), face1.TxInterests[0].GetHopLimit())

	// Data satisfies Interest
	data := makeData("/A/1")
	copyPitToken(data, face1.TxInterests[0])
	face1.Rx(data)
	time.Sleep(STEP_DELAY)
	assert.Len(face3.TxData, 1)
	// whether face3 receives Data or not is unspecified

	// HopLimit reaches zero, can still retrieve from CS
	interest1a := makeInterest("/A/1", uint8(1))
	face4.Rx(interest1a)
	time.Sleep(STEP_DELAY)
	assert.Len(face4.TxData, 1)
}

func TestCsHit(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	interestB1 := makeInterest("/B/1")
	setPitToken(interestB1, 0x193d673cdb9f85ac)
	face1.Rx(interestB1)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)

	dataB1 := makeData("/B/1")
	copyPitToken(dataB1, face2.TxInterests[0])
	face2.Rx(dataB1)
	time.Sleep(STEP_DELAY)
	require.Len(face1.TxData, 1)
	assert.Equal(uint64(0x193d673cdb9f85ac), getPitToken(face1.TxData[0]))
	assert.Equal(0*time.Millisecond, face1.TxData[0].GetFreshnessPeriod())

	interestB1mbf := makeInterest("/B/1", ndn.MustBeFreshFlag)
	setPitToken(interestB1mbf, 0xf716737325e04a77)
	face1.Rx(interestB1mbf)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 2)

	dataB1fp := makeData("/B/1", 2500*time.Millisecond)
	copyPitToken(dataB1fp, face2.TxInterests[1])
	face2.Rx(dataB1fp)
	time.Sleep(STEP_DELAY)
	require.Len(face1.TxData, 2)
	assert.Equal(uint64(0xf716737325e04a77), getPitToken(face1.TxData[1]))
	assert.Equal(2500*time.Millisecond, face1.TxData[1].GetFreshnessPeriod())

	interestB1 = makeInterest("/B/1")
	setPitToken(interestB1, 0xaec62dad2f669e6b)
	face1.Rx(interestB1)
	time.Sleep(STEP_DELAY)
	assert.Len(face2.TxInterests, 2)
	require.Len(face1.TxData, 3)
	assert.Equal(uint64(0xaec62dad2f669e6b), getPitToken(face1.TxData[2]))
	assert.Equal(2500*time.Millisecond, face1.TxData[2].GetFreshnessPeriod())

	interestB1mbf = makeInterest("/B/1", ndn.MustBeFreshFlag)
	setPitToken(interestB1mbf, 0xb5565a4e715c858d)
	face1.Rx(interestB1mbf)
	time.Sleep(STEP_DELAY)
	assert.Len(face2.TxInterests, 2)
	require.Len(face1.TxData, 4)
	assert.Equal(uint64(0xb5565a4e715c858d), getPitToken(face1.TxData[3]))
	assert.Equal(2500*time.Millisecond, face1.TxData[3].GetFreshnessPeriod())

	fibCnt := fixture.ReadFibCounters("/B")
	assert.Equal(uint64(4), fibCnt.NRxInterests)
	assert.Equal(uint64(2), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(2), fibCnt.NTxInterests)
}

func TestFwHint(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	face4 := fixture.CreateFace()
	face5 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "multicast", face1.GetFaceId())
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())
	fixture.SetFibEntry("/C", "multicast", face3.GetFaceId())

	interest1 := makeInterest("/A/1", ndn.FHDelegation{1, "/B"}, ndn.FHDelegation{2, "/C"})
	setPitToken(interest1, 0x5c2fc6c972d830e7)
	face4.Rx(interest1)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 0)
	assert.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 0)

	interest2 := makeInterest("/A/1", ndn.FHDelegation{1, "/C"}, ndn.FHDelegation{2, "/B"})
	setPitToken(interest2, 0x52e61e9eee7025b7)
	face5.Rx(interest2)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 0)
	require.Len(face2.TxInterests, 1)
	assert.Len(face3.TxInterests, 1)

	interest3 := makeInterest("/A/1", ndn.FHDelegation{1, "/Z"}, ndn.FHDelegation{2, "/B"})
	setPitToken(interest3, 0xa4291e2123c8211e)
	face5.Rx(interest3)
	time.Sleep(STEP_DELAY)
	assert.Len(face1.TxInterests, 0)
	assert.True(len(face2.TxInterests) <= 2)
	require.Len(face2.TxInterests, 2)
	assert.Equal(getPitToken(face2.TxInterests[0]), getPitToken(face2.TxInterests[1]))
	require.Len(face3.TxInterests, 1)

	data1 := makeData("/A/1", 1*time.Second) // satisfies interest1 and interest3
	copyPitToken(data1, face2.TxInterests[0])
	face2.Rx(data1)
	time.Sleep(STEP_DELAY)
	require.Len(face4.TxData, 1)
	assert.Equal(uint64(0x5c2fc6c972d830e7), getPitToken(face4.TxData[0]))
	assert.Equal(1*time.Second, face4.TxData[0].GetFreshnessPeriod())
	require.Len(face5.TxData, 1)
	assert.Equal(uint64(0xa4291e2123c8211e), getPitToken(face5.TxData[0]))
	assert.Equal(1*time.Second, face5.TxData[0].GetFreshnessPeriod())

	data2 := makeData("/A/1", 2*time.Second) // satisfies interest2
	copyPitToken(data2, face3.TxInterests[0])
	face3.Rx(data2)
	time.Sleep(STEP_DELAY)
	require.Len(face5.TxData, 2)
	assert.Equal(uint64(0x52e61e9eee7025b7), getPitToken(face5.TxData[1]))
	assert.Equal(2*time.Second, face5.TxData[1].GetFreshnessPeriod())

	interest4 := makeInterest("/A/1", ndn.FHDelegation{1, "/C"}) // matches data2
	setPitToken(interest4, 0xbb19e173f937f221)
	face4.Rx(interest4)
	time.Sleep(STEP_DELAY)
	require.Len(face4.TxData, 2)
	assert.Equal(uint64(0xbb19e173f937f221), getPitToken(face4.TxData[1]))
	assert.Equal(2*time.Second, face4.TxData[1].GetFreshnessPeriod())

	fibCnt := fixture.ReadFibCounters("/A")
	assert.Equal(uint64(0), fibCnt.NRxInterests)
	assert.Equal(uint64(0), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(0), fibCnt.NTxInterests)
	fibCnt = fixture.ReadFibCounters("/B")
	assert.Equal(uint64(2), fibCnt.NRxInterests)
	assert.Equal(uint64(1), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(2), fibCnt.NTxInterests)
	fibCnt = fixture.ReadFibCounters("/C")
	assert.Equal(uint64(2), fibCnt.NRxInterests)
	assert.Equal(uint64(1), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(1), fibCnt.NTxInterests)
}

func TestImplicitDigest(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/B", "multicast", face2.GetFaceId())

	dataB1 := makeData("/B/1")
	fullNameB1 := dataB1.GetFullName().String()

	interestB1 := makeInterest(fullNameB1)
	setPitToken(interestB1, 0xce2e9bce22327e97)
	face1.Rx(interestB1)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 1)

	copyPitToken(dataB1, face2.TxInterests[0])
	face2.Rx(dataB1)
	time.Sleep(STEP_DELAY)
	require.Len(face1.TxData, 1)
	assert.Equal(uint64(0xce2e9bce22327e97), getPitToken(face1.TxData[0]))

	interestB1 = makeInterest(fullNameB1)
	setPitToken(interestB1, 0x5446c548dd1a5c89)
	face1.Rx(interestB1)
	time.Sleep(STEP_DELAY)
	assert.Len(face2.TxInterests, 1)

	// CS hit
	require.Len(face1.TxData, 2)
	assert.Equal(uint64(0x5446c548dd1a5c89), getPitToken(face1.TxData[1]))

	fibCnt := fixture.ReadFibCounters("/B")
	assert.Equal(uint64(2), fibCnt.NRxInterests)
	assert.Equal(uint64(1), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(1), fibCnt.NTxInterests)

	// /B/2 is fragmented, which is not supported in some cryptodev
	dataB2 := makeData("/B/2", ndn.TlvBytes(bytes.Repeat([]byte{0xC0}, 300)))
	fullNameB2orig := dataB2.GetFullName().String()
	mbuftestenv.PacketSplitTailSegment(dataB2.GetPacket().AsMbuf(), 5)
	fullNameB2 := dataB2.GetFullName().String()
	assert.Equal(fullNameB2orig, fullNameB2)

	interestB2 := makeInterest(fullNameB2)
	setPitToken(interestB2, 0x02a0f62d1828a80c)
	face1.Rx(interestB2)
	time.Sleep(STEP_DELAY)
	require.Len(face2.TxInterests, 2)

	copyPitToken(dataB2, face2.TxInterests[1])
	face2.Rx(dataB2)
	time.Sleep(STEP_DELAY)
	require.Len(face1.TxData, 3)
	assert.Equal(uint64(0x02a0f62d1828a80c), getPitToken(face1.TxData[2]))
}
