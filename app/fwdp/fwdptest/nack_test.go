package fwdptest

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/iface/intface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

func TestNackMerge(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3 := intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	fixture.SetFibEntry("/A", "multicast", face2.ID, face3.ID)

	// Interest is forwarded to two upstream nodes
	face1.Tx <- ndn.MakeInterest("/A/1", ndn.NonceFromUint(0x2ea29515), lphToken(0xf3fb4ef802d3a9d3))
	fixture.StepDelay()
	assert.Equal(1, collect2.Count())
	assert.Equal(1, collect3.Count())

	// Nack from first upstream, no action
	face2.Tx <- ndn.MakeNack(collect2.Get(-1).Interest, an.NackNoRoute)
	fixture.StepDelay()
	assert.Equal(0, collect1.Count())

	// Nack again from first upstream, no action
	face2.Tx <- ndn.MakeNack(collect2.Get(-1).Interest, an.NackNoRoute)
	fixture.StepDelay()
	assert.Equal(0, collect1.Count())

	// Nack from second upstream, Nack to downstream
	face3.Tx <- ndn.MakeNack(collect3.Get(-1).Interest, an.NackCongestion)
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())

	if packet := collect1.Get(-1); assert.NotNil(packet.Nack) {
		assert.EqualValues(an.NackCongestion, packet.Nack.Reason)
		assert.Equal(ndn.NonceFromUint(0x2ea29515), packet.Nack.Interest.Nonce)
		assert.Equal(ndn.PitTokenFromUint(0xf3fb4ef802d3a9d3), packet.Lp.PitToken)
	}

	// Data from first upstream, should not reach downstream because PIT entry is gone
	face2.Tx <- ndn.MakeData(collect2.Get(-1).Interest)
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
}

func TestNackDuplicate(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2, face3 := intface.MustNew(), intface.MustNew(), intface.MustNew()
	collect1, collect2, collect3 := intface.Collect(face1), intface.Collect(face2), intface.Collect(face3)
	fixture.SetFibEntry("/A", "multicast", face3.ID)

	// two Interests come from two downstream nodes
	face1.Tx <- ndn.MakeInterest("/A/1", ndn.NonceFromUint(0x2ea29515))
	face2.Tx <- ndn.MakeInterest("/A/1", ndn.NonceFromUint(0xc33b0c68))
	fixture.StepDelay()
	assert.Equal(1, collect3.Count())

	// upstream node returns Nack against first Interest
	// forwarder should resend Interest with another nonce
	interest0 := collect3.Get(0).Interest
	face3.Tx <- ndn.MakeNack(interest0, an.NackDuplicate)
	fixture.StepDelay()
	assert.Equal(2, collect3.Count())
	interest1 := collect3.Get(1).Interest
	assert.NotEqual(interest0.Nonce, interest1.Nonce)
	assert.Equal(0, collect1.Count())
	assert.Equal(0, collect2.Count())

	// upstream node returns Nack against second Interest as well
	// forwarder should return Nack to downstream
	face3.Tx <- ndn.MakeNack(interest1, an.NackDuplicate)
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.Equal(1, collect2.Count())

	fibCnt := fixture.ReadFibCounters("/A")
	assert.Equal(uint64(2), fibCnt.NRxInterests)
	assert.Equal(uint64(0), fibCnt.NRxData)
	assert.Equal(uint64(2), fibCnt.NRxNacks)
	assert.Equal(uint64(2), fibCnt.NTxInterests)
}

func TestReturnNacks(t *testing.T) {
	assert, _ := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1, face2 := intface.MustNew(), intface.MustNew()
	collect1 := intface.Collect(face1)
	fixture.SetFibEntry("/A", "reject", face2.ID)

	face1.Tx <- ndn.MakeInterest("/A/1", ndn.NonceFromUint(0x2ea29515))
	fixture.StepDelay()
	assert.Equal(1, collect1.Count())
	assert.NotNil(collect1.Get(-1).Nack)
}
