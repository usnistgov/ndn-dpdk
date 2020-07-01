package pktmbuf_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf/mbuftestenv"
)

func TestPktItZero(t *testing.T) {
	assert, _ := makeAR(t)

	var pi pktmbuf.PacketIterator
	assert.True(pi.IsEnd())
}

func TestPktItAdvanceDistance(t *testing.T) {
	assert, require := makeAR(t)

	pkt := mbuftestenv.MakePacket("", "A0A1A2A3", "B0B1B2", "", "C0C1", "D0")
	defer pkt.Close()
	const pktlen = 10
	require.Equal(pktlen, pkt.Len())

	var pi1 [pktlen]pktmbuf.PacketIterator
	var pi2 [pktlen]pktmbuf.PacketIterator

	for step := 1; step < pktlen; step++ {
		for i := 0; i < pktlen; i += step {
			if i == 0 {
				pi2[i] = pktmbuf.NewPacketIterator(pkt)
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

	pkt := makePacket("", "A0A1A2A3", "B0B1B2", "", "C0C1", "D0")
	defer pkt.Close()
	const pktlen = 10
	require.Equal(pktlen, pkt.Len())
	payload := bytesFromHex("A0A1A2A3B0B1B2C0C1D0")

	for offset := 0; offset <= pktlen; offset++ {
		for count := 1; count < pktlen-offset; count++ {
			pi := pktmbuf.NewPacketIterator(pkt)
			pi.Advance(offset)
			clone, e := pi.MakeIndirect(count, mbuftestenv.Indirect.Pool())
			if !assert.NoError(e, "%d-%d", offset, count) {
				continue
			}
			assert.Equal(count, clone.Len())
			assert.Equal(payload[offset:offset+count], clone.ReadAll(), "%d-%d", offset, count)
			for segIndex, segLen := range mbuftestenv.ListSegmentLengths(clone) {
				assert.NotZero(segLen, "%d-%d:%d", offset, count, segIndex)
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
