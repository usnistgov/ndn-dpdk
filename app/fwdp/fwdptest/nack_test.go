package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestNackMerge(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "multicast", face2.GetFaceId(), face3.GetFaceId())

	// Interest is forwarded to two upstream nodes
	interest := ndntestutil.MakeInterest("/A/1", uint32(0x2ea29515))
	ndntestutil.SetPitToken(interest, 0xf3fb4ef802d3a9d3)
	face1.Rx(interest)
	time.Sleep(100 * time.Millisecond)
	require.Len(face2.TxInterests, 1)
	require.Len(face3.TxInterests, 1)

	// Nack from first upstream, no action
	nack2 := ndn.MakeNackFromInterest(ndntestutil.MakeInterest("/A/1", uint32(0x2ea29515)), ndn.NackReason_NoRoute)
	ndntestutil.CopyPitToken(nack2, face2.TxInterests[0])
	face2.Rx(nack2)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxNacks, 0)

	// Nack again from first upstream, no action
	nack2 = ndn.MakeNackFromInterest(ndntestutil.MakeInterest("/A/1", uint32(0x2ea29515)), ndn.NackReason_NoRoute)
	ndntestutil.CopyPitToken(nack2, face2.TxInterests[0])
	face2.Rx(nack2)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxNacks, 0)

	// Nack from second upstream, Nack to downstream
	nack3 := ndn.MakeNackFromInterest(ndntestutil.MakeInterest("/A/1", uint32(0x2ea29515)), ndn.NackReason_Congestion)
	ndntestutil.CopyPitToken(nack3, face3.TxInterests[0])
	face3.Rx(nack3)
	time.Sleep(100 * time.Millisecond)
	require.Len(face1.TxNacks, 1)

	nack1 := face1.TxNacks[0]
	assert.Equal(ndn.NackReason_Congestion, nack1.GetReason())
	assert.Equal(uint32(0x2ea29515), nack1.GetInterest().GetNonce())
	assert.Equal(uint64(0xf3fb4ef802d3a9d3), ndntestutil.GetPitToken(nack1))

	// Data from first upstream, should not reach downstream because PIT entry is gone
	data2 := ndntestutil.MakeData("/A/1")
	ndntestutil.CopyPitToken(data2, face2.TxInterests[0])
	face2.Rx(data2)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 0)
}

func TestNackDuplicate(t *testing.T) {
	assert, require := makeAR(t)
	fixture := NewFixture(t)
	defer fixture.Close()

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	face3 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "multicast", face3.GetFaceId())

	// two Interests come from two downstream nodes
	interest1 := ndntestutil.MakeInterest("/A/1", uint32(0x2ea29515))
	face1.Rx(interest1)
	interest2 := ndntestutil.MakeInterest("/A/1", uint32(0xc33b0c68))
	face2.Rx(interest2)
	time.Sleep(10 * time.Millisecond)
	require.Len(face3.TxInterests, 1)

	// upstream node returns Nack against first Interest
	// forwarder should resend Interest with another nonce
	nonce1 := face3.TxInterests[0].GetNonce()
	nack1 := ndn.MakeNackFromInterest(face3.TxInterests[0], ndn.NackReason_Duplicate)
	face3.Rx(nack1)
	time.Sleep(10 * time.Millisecond)
	require.Len(face3.TxInterests, 2)
	nonce2 := face3.TxInterests[1].GetNonce()
	assert.NotEqual(nonce1, nonce2)
	assert.Len(face1.TxNacks, 0)
	assert.Len(face2.TxNacks, 0)

	// upstream node returns Nack against second Interest as well
	// forwarder should return Nack to downstream
	nack2 := ndn.MakeNackFromInterest(face3.TxInterests[1], ndn.NackReason_Duplicate)
	face3.Rx(nack2)
	time.Sleep(10 * time.Millisecond)
	assert.Len(face1.TxNacks, 1)
	assert.Len(face2.TxNacks, 1)

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

	face1 := fixture.CreateFace()
	face2 := fixture.CreateFace()
	fixture.SetFibEntry("/A", "reject", face2.GetFaceId())

	interest1 := ndntestutil.MakeInterest("/A/1", uint32(0x2ea29515))
	face1.Rx(interest1)
	time.Sleep(10 * time.Millisecond)
	assert.Len(face1.TxNacks, 1)
}
