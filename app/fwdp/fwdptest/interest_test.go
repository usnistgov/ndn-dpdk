package fwdptest

import (
	"bytes"
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/app/fwdp"
	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func TestInterestData(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3 := intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	fixture.SetFibEntry("/B", "multicast", face2.ID)
	fixture.SetFibEntry("/C", "multicast", face3.ID)

	face1.Tx <- ndn.MakeInterest("/B/1", lphToken(0x0290dd7089e9d790))
	fixture.StepDelay()
	assert.Equal(1, collect2.Count())
	assert.Equal(0, collect3.Count())

	face2.Tx <- ndn.MakeData(collect2.Get(-1).Interest)
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0x0290dd7089e9d790), ndn.PitTokenToUint(packet.Lp.PitToken))
	}

	fibCnt := fixture.ReadFibCounters("/B")
	assert.Equal(uint64(1), fibCnt.NRxInterests)
	assert.Equal(uint64(1), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(1), fibCnt.NTxInterests)
}

func TestInterestDupNonce(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3 := intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	fixture.SetFibEntry("/A", "multicast", face3.ID)

	face1.Tx <- ndn.MakeInterest("/A/1", ndn.NonceFromUint(0x6f937a51), lphToken(0x3bddf54cffbc6ad0))
	fixture.StepDelay()
	assert.Equal(1, collect3.Count())

	face2.Tx <- ndn.MakeInterest("/A/1", ndn.NonceFromUint(0x6f937a51), lphToken(0x3bddf54cffbc6ad0))
	fixture.StepDelay()
	assert.Equal(1, collect3.Count())
	assert.Equal(1, collect2.Count())
	if packet := collect2.Get(-1); assert.NotNil(packet.Nack) {
		assert.EqualValues(an.NackDuplicate, packet.Nack.Reason)
		assert.Equal(uint64(0x3bddf54cffbc6ad0), ndn.PitTokenToUint(packet.Lp.PitToken))
	}
	assert.Equal(uint64(1), fixture.SumCounter(func(fwd *fwdp.Fwd) uint64 {
		return fwd.Counters().NDupNonce
	}))

	collect1.Clear()
	collect2.Clear()
	face3.Tx <- ndn.MakeData(collect3.Get(-1).Interest)
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.NotNil(collect1.Get(-1).Data)
	assert.Equal(0, collect2.Count())
}

func TestInterestSuppress(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3 := intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect3 := intface.Collect(face3)
	fixture.SetFibEntry("/A", "multicast", face3.ID)

	go func() {
		ticker := time.NewTicker(1 * time.Millisecond)
		defer ticker.Stop()
		for i := 0; i < 400; i++ {
			<-ticker.C
			interest := ndn.MakeInterest("/A/1", lphToken(0xf4aab9f23eb5271e^uint64(i)))
			if i%2 == 0 {
				face1.Tx <- interest
			} else {
				face2.Tx <- interest
			}
		}
	}()

	time.Sleep(500 * time.Millisecond)
	assert.InDelta(7, collect3.Count(), 1)
	// suppression config is min=10, multiplier=2, max=100,
	// so 7 Interests should be forwarded at 0, 10, 30, 70, 150, 250, 350,
	// but this could be off by one on a slower machine.
}

func TestInterestNoRoute(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := intface.MustNew()
	collect1 := intface.Collect(face1)

	face1.Tx <- ndn.MakeInterest("/A/1", lphToken(0x431328d8b4075167))
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Nack) {
		assert.Equal(uint64(0x431328d8b4075167), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.EqualValues(an.NackNoRoute, packet.Nack.Reason)
	}
	assert.Equal(uint64(1), fixture.SumCounter(func(fwd *fwdp.Fwd) uint64 {
		return fwd.Counters().NNoFibMatch
	}))
}

func TestHopLimit(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3, face4 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect3, collect4 := intface.Collect(face1), intface.Collect(face3), intface.Collect(face4)
	fixture.SetFibEntry("/A", "multicast", face1.ID)

	// cannot test HopLimit=0 because it's rejected by decoder,
	// so MakeInterest would fail

	// HopLimit becomes zero, cannot forward
	face2.Tx <- ndn.MakeInterest("/A/1", ndn.HopLimit(1))
	fixture.StepDelay()
	assert.Equal(0, collect1.Count())

	// HopLimit is 1 after decrementing, forwarded with HopLimit=1
	face3.Tx <- ndn.MakeInterest("/A/1", ndn.HopLimit(2))
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.EqualValues(1, collect1.Get(-1).Interest.HopLimit)

	// Data satisfies Interest
	face1.Tx <- ndn.MakeData(collect1.Get(-1).Interest)
	fixture.StepDelay()
	assert.Equal(1, collect3.Count())
	// whether face3 receives Data or not is unspecified

	// HopLimit reaches zero, can still retrieve from CS
	face4.Tx <- ndn.MakeInterest("/A/1", ndn.HopLimit(1))
	fixture.StepDelay()
	assert.Equal(1, collect4.Count())
}

func TestCsHit(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	face1.Tx <- ndn.MakeInterest("/B/1", lphToken(0x193d673cdb9f85ac))
	fixture.StepDelay()
	assert.Equal(1, collect2.Count())

	face2.Tx <- ndn.MakeData(collect2.Get(-1).Interest)
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0x193d673cdb9f85ac), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(0*time.Millisecond, packet.Data.Freshness)
	}

	face1.Tx <- ndn.MakeInterest("/B/1", ndn.MustBeFreshFlag, lphToken(0xf716737325e04a77))
	fixture.StepDelay()
	assert.Equal(2, collect2.Count())

	face2.Tx <- ndn.MakeData(collect2.Get(-1).Interest, 2500*time.Millisecond)
	fixture.StepDelay()
	assert.Equal(2, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0xf716737325e04a77), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(2500*time.Millisecond, packet.Data.Freshness)
	}

	face1.Tx <- ndn.MakeInterest("/B/1", lphToken(0xaec62dad2f669e6b))
	fixture.StepDelay()
	assert.Equal(2, collect2.Count())
	assert.Equal(3, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0xaec62dad2f669e6b), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(2500*time.Millisecond, packet.Data.Freshness)
	}

	face1.Tx <- ndn.MakeInterest("/B/1", ndn.MustBeFreshFlag, lphToken(0xb5565a4e715c858d))
	fixture.StepDelay()
	assert.Equal(2, collect2.Count())
	assert.Equal(4, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0xb5565a4e715c858d), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(2500*time.Millisecond, packet.Data.Freshness)
	}

	fibCnt := fixture.ReadFibCounters("/B")
	assert.Equal(uint64(4), fibCnt.NRxInterests)
	assert.Equal(uint64(2), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(2), fibCnt.NTxInterests)
}

func TestFwHint(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3, face4, face5 := intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3, collect4, collect5 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3), intface.Collect(face4), intface.Collect(face5)
	fixture.SetFibEntry("/A", "multicast", face1.ID)
	fixture.SetFibEntry("/B", "multicast", face2.ID)
	fixture.SetFibEntry("/C", "multicast", face3.ID)

	face4.Tx <- ndn.MakeInterest("/A/1", ndn.MakeFHDelegation(1, "/B"), ndn.MakeFHDelegation(2, "/C"), lphToken(0x5c2fc6c972d830e7))
	fixture.StepDelay()
	assert.Equal(0, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(0, collect3.Count())

	face5.Tx <- ndn.MakeInterest("/A/1", ndn.MakeFHDelegation(1, "/C"), ndn.MakeFHDelegation(2, "/B"), lphToken(0x52e61e9eee7025b7))
	fixture.StepDelay()
	assert.Equal(0, collect1.Count())
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	face5.Tx <- ndn.MakeInterest("/A/1", ndn.MakeFHDelegation(1, "/Z"), ndn.MakeFHDelegation(2, "/B"), lphToken(0xa4291e2123c8211e))
	fixture.StepDelay()
	assert.Equal(0, collect1.Count())
	assert.Equal(2, collect2.Count())
	assert.Equal(1, collect3.Count())
	assert.Equal(collect2.Get(0).Lp.PitToken, collect2.Get(1).Lp.PitToken)

	face2.Tx <- ndn.MakeData(collect2.Get(0).Interest, 1*time.Second) // satisfies first and third Interests
	fixture.StepDelay()
	assert.Equal(1, collect4.Count())
	if packet := collect4.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0x5c2fc6c972d830e7), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(1*time.Second, packet.Data.Freshness)
	}
	assert.Equal(1, collect5.Count())
	if packet := collect5.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0xa4291e2123c8211e), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(1*time.Second, packet.Data.Freshness)
	}

	face3.Tx <- ndn.MakeData(collect3.Get(-1).Interest, 2*time.Second) // satisfies second Interest
	fixture.StepDelay()
	assert.Equal(2, collect5.Count())
	if packet := collect5.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0x52e61e9eee7025b7), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(2*time.Second, packet.Data.Freshness)
	}

	face4.Tx <- ndn.MakeInterest("/A/1", ndn.MakeFHDelegation(1, "/C"), lphToken(0xbb19e173f937f221)) // matches second Data
	fixture.StepDelay()
	assert.Equal(2, collect4.Count())
	if packet := collect4.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0xbb19e173f937f221), ndn.PitTokenToUint(packet.Lp.PitToken))
		assert.Equal(2*time.Second, packet.Data.Freshness)
	}

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

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	data := ndn.MakeData("/B/1")
	fullName := data.FullName()

	face1.Tx <- ndn.MakeInterest(fullName, lphToken(0xce2e9bce22327e97))
	fixture.StepDelay()
	assert.Equal(1, collect2.Count())

	packet := data.ToPacket()
	packet.Lp.PitToken = collect2.Get(-1).Lp.PitToken
	face2.Tx <- packet
	fixture.StepDelay()
	require.Equal(1, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0xce2e9bce22327e97), ndn.PitTokenToUint(packet.Lp.PitToken))
	}

	face1.Tx <- ndn.MakeInterest(fullName, lphToken(0x5446c548dd1a5c89))
	fixture.StepDelay()
	assert.Equal(1, collect2.Count())

	// CS hit
	require.Equal(2, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0x5446c548dd1a5c89), ndn.PitTokenToUint(packet.Lp.PitToken))
	}

	fibCnt := fixture.ReadFibCounters("/B")
	assert.Equal(uint64(2), fibCnt.NRxInterests)
	assert.Equal(uint64(1), fibCnt.NRxData)
	assert.Equal(uint64(0), fibCnt.NRxNacks)
	assert.Equal(uint64(1), fibCnt.NTxInterests)
}

func TestImplicitDigestFragmented(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1, collect2 := intface.Collect(face1), intface.Collect(face2)
	fixture.SetFibEntry("/B", "multicast", face2.ID)

	// /B/2 is fragmented, which is not supported in some cryptodev
	data := ndn.MakeData("/B/2", bytes.Repeat([]byte{0xC0}, 300))
	fullName := data.FullName()

	face1.Tx <- ndn.MakeInterest(fullName, lphToken(0x02a0f62d1828a80c))
	fixture.StepDelay()
	assert.Equal(1, collect2.Count())

	packet := data.ToPacket()
	packet.Lp.PitToken = collect2.Get(-1).Lp.PitToken
	frags, e := ndn.NewLpFragmenter(200).Fragment(data.ToPacket())
	require.NoError(e)
	require.Greater(len(frags), 1)
	for _, frag := range frags {
		face2.Tx <- frag
		fixture.StepDelay()
	}
	assert.Equal(1, collect1.Count())
	if packet := collect1.Get(-1); assert.NotNil(packet.Data) {
		assert.Equal(uint64(0x02a0f62d1828a80c), ndn.PitTokenToUint(packet.Lp.PitToken))
	}
}
