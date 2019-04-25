package dpdktest

import (
	"testing"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestPktItEmpty(t *testing.T) {
	assert, _ := makeAR(t)

	var pi dpdk.PacketIterator
	assert.True(pi.IsEnd())
}

func TestPktItAdvanceDistance(t *testing.T) {
	assert, require := makeAR(t)
	dpdktestenv.MakeDirectMp(63, 0, 4)

	pkt := dpdktestenv.PacketFromHex("", "A0A1A2A3", "B0B1B2", "", "C0C1", "D0")
	defer pkt.Close()
	const pktlen = 10
	require.Equal(pktlen, pkt.Len())

	var pi1 [pktlen]dpdk.PacketIterator
	var pi2 [pktlen]dpdk.PacketIterator

	for step := 1; step < pktlen; step++ {
		for i := 0; i < pktlen; i += step {
			if i == 0 {
				pi2[i] = dpdk.NewPacketIterator(pkt)
				assert.Equal(0, pi2[i].Advance(0), "%d-%d", step, i)
			} else {
				pi2[i] = pi2[i-step]
				assert.Equal(step, pi2[i].Advance(step), "%d-%d", step, i)
			}

			if step == 1 {
				pi1[i] = pi2[i]
			} else {
				assert.True(pi1[i] == pi2[i], "%d-%d", step, i)
			}
		}
	}
}

func TestPktItMakeIndirect(t *testing.T) {
	assert, require := makeAR(t)
	dpdktestenv.MakeDirectMp(63, 0, 4)
	mpi := dpdktestenv.MakeIndirectMp(63)

	pkt := dpdktestenv.PacketFromHex("", "A0A1A2A3", "B0B1B2", "", "C0C1", "D0")
	defer pkt.Close()
	const pktlen = 10
	require.Equal(pktlen, pkt.Len())
	payload := dpdktestenv.BytesFromHex("A0A1A2A3B0B1B2C0C1D0")

	for offset := 0; offset <= pktlen; offset++ {
		for count := 1; count < pktlen-offset; count++ {
			pi := dpdk.NewPacketIterator(pkt)
			pi.Advance(offset)
			clone, e := pi.MakeIndirect(count, mpi)
			if !assert.NoError(e, "%d-%d", offset, count) {
				continue
			}
			assert.Equal(count, clone.Len())
			assert.Equal(payload[offset:offset+count], clone.ReadAll(), "%d-%d", offset, count)
			for seg, ok := clone.GetFirstSegment(), true; ok; seg, ok = seg.GetNext() {
				assert.NotZero(seg.Len())
			}
			clone.Close()

			if offset+count < pktlen {
				assert.Equal(payload[offset+count], byte(pi.PeekOctet()))
			} else {
				assert.Equal(-1, pi.PeekOctet())
			}
		}
	}
}
