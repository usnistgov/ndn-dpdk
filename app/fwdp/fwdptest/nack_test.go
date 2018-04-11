package fwdptest

import (
	"testing"
	"time"

	"ndn-dpdk/app/fwdp/fwdptestfixture"
	"ndn-dpdk/ndn"
	"ndn-dpdk/ndn/ndntestutil"
)

func TestNackMerge(t *testing.T) {
	assert, require := makeAR(t)
	fixture := fwdptestfixture.New(t)
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
	assert.Equal(nack1.GetReason(), ndn.NackReason_Congestion)
	assert.Equal(nack1.GetInterest().GetNonce(), uint32(0x2ea29515))
	assert.Equal(ndntestutil.GetPitToken(nack1), uint64(0xf3fb4ef802d3a9d3))

	// Data from first upstream, should not reach downstream because PIT entry is gone
	data2 := ndntestutil.MakeData("/A/1")
	ndntestutil.CopyPitToken(data2, face2.TxInterests[0])
	face2.Rx(data2)
	time.Sleep(100 * time.Millisecond)
	assert.Len(face1.TxData, 0)
}
