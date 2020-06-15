package ndn_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

func TestLpHeaderDecode(t *testing.T) {
	assert, _ := makeAR(t)

	const (
		bareInterest     = "0505 0703080141"
		payloadInterest  = "5007 " + bareInterest
		payloadInterestL = 7
		payloadFragment  = "5004 D0D1D2D3"
		payloadFragmentL = 4
	)

	tests := []struct {
		input      string
		bad        bool
		seqNum     uint64
		fragIndex  uint16
		fragCount  uint16
		pitToken   uint64
		nackReason ndn.NackReason
		congMark   ndn.CongMark
		payloadL   int
	}{
		{input: "", bad: true},
		{input: bareInterest, fragCount: 1, payloadL: payloadInterestL},
		{input: "6409 payload=" + payloadInterest, fragCount: 1, payloadL: payloadInterestL},
		{input: "6402 unknown-critical=6300", bad: true},
		{input: "6404 unknown-critical=FD03BF00", bad: true},
		{input: "6404 unknown-ignored=FD03BC00", fragCount: 1},
		{input: "6413 seq=5108A0A1A2A3A4A5A600 fragcount=530102 payload=" + payloadFragment,
			seqNum: 0xA0A1A2A3A4A5A600, fragIndex: 0, fragCount: 2,
			payloadL: payloadFragmentL},
		{input: "6416 seq=5108A0A1A2A3A4A5A601 fragindex=520101 fragcount=530102 " +
			"payload=" + payloadFragment,
			seqNum: 0xA0A1A2A3A4A5A601, fragIndex: 1, fragCount: 2,
			payloadL: payloadFragmentL},
		{input: "6417 seq=5108A0A1A2A3A4A5A601 fragindex=520102 fragcount=530102 " +
			"payload=" + payloadFragment, bad: true}, // FragIndex >= FragCount
		{input: "6413 pittoken=62089A414B412BC38EB2 payload=" + payloadInterest,
			fragCount: 1, pitToken: 0xB28EC32B414B419A, payloadL: payloadInterestL},
		{input: "6406 pittoken=620420A3C0D7", bad: true}, // PitToken is not 8-octet
		{input: "640D nack=FD032000(noreason) payload=" + payloadInterest,
			fragCount: 1, nackReason: ndn.NackReason_Unspecified, payloadL: payloadInterestL},
		{input: "6412 nack=FD032005(FD03210196~noroute) payload=" + payloadInterest,
			fragCount: 1, nackReason: ndn.NackReason_NoRoute, payloadL: payloadInterestL},
		{input: "640E congmark=FD03400104 payload=" + payloadInterest,
			fragCount: 1, congMark: 4, payloadL: payloadInterestL},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsMbuf().Close()
		e := pkt.ParseL2()
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			if !assert.Equal(ndn.L2PktType_NdnlpV2, pkt.GetL2Type(), tt.input) {
				continue
			}
			lph := pkt.GetLpHdr()
			seqNum, fragIndex, fragCount := lph.GetFragFields()
			assert.Equal(tt.seqNum, seqNum, tt.input)
			assert.Equal(tt.fragIndex, fragIndex, tt.input)
			assert.Equal(tt.fragCount, fragCount, tt.input)
			assert.Equal(tt.pitToken, lph.GetPitToken(), tt.input)
			assert.Equal(tt.nackReason, lph.GetNackReason(), tt.input)
			assert.Equal(tt.congMark, lph.GetCongMark(), tt.input)
			assert.Equal(tt.payloadL, pkt.AsMbuf().Len(), tt.input)
		}
	}
}
