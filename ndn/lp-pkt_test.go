package ndn

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestLpPkt(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input      string
		bad        bool
		hasPayload bool
		seqNo      uint64
		fragIndex  uint16
		fragCount  uint16
		pitToken   uint64
		nackReason NackReason
		congMark   CongMark
	}{
		{input: "", bad: true},
		{input: "6406 payload=5004D0D1D2D3", hasPayload: true, fragCount: 1},
		{input: "6402 unknown-critical=6300", bad: true},
		{input: "6404 unknown-critical=FD03BF00", bad: true},
		{input: "6404 unknown-ignored=FD03BC00", fragCount: 1},
		{input: "6411 seq=5108A0A1A2A3A4A5A600 fragcount=530102 payload=5002D0D1",
			hasPayload: true, seqNo: 0xA0A1A2A3A4A5A600, fragIndex: 0, fragCount: 2},
		{input: "6414 seq=5108A0A1A2A3A4A5A601 fragindex=520101 fragcount=530102 payload=5002D2D3",
			hasPayload: true, seqNo: 0xA0A1A2A3A4A5A601, fragIndex: 1, fragCount: 2},
		{input: "6417 seq=5108A0A1A2A3A4A5A601 fragindex=520102 fragcount=530102 payload=5002D2D3",
			bad: true}, // FragIndex>=FragCount
		{input: "640A pittoken=62089A414B412BC38EB2",
			fragCount: 1, pitToken: 0xB28EC32B414B419A},
		{input: "6406 pittoken=620420A3C0D7", bad: true}, // only accept 8-octet PitToken
		{input: "6404 nack=FD032000(noreason)",
			fragCount: 1, nackReason: NackReason_Unspecified},
		{input: "6409 nack=FD032005(FD03210196~noroute)",
			fragCount: 1, nackReason: NackReason_NoRoute},
		{input: "6405 congmark=FD03400104", fragCount: 1, congMark: 4},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		lpp, e := d.ReadLpPkt()
		if tt.bad {
			assert.Error(e, tt.input)
		} else {
			if assert.NoError(e, tt.input) {
				assert.Equal(tt.hasPayload, lpp.HasPayload(), tt.input)
				assert.Equal(tt.fragCount > 1, lpp.IsFragmented(), tt.input)
				seqNo, fragIndex, fragCount := lpp.GetFragFields()
				assert.Equal(tt.seqNo, seqNo, tt.input)
				assert.Equal(tt.fragIndex, fragIndex, tt.input)
				assert.Equal(tt.fragCount, fragCount, tt.input)
				assert.Equal(tt.pitToken, lpp.GetPitToken(), tt.input)
				assert.Equal(tt.nackReason, lpp.GetNackReason(), tt.input)
				assert.Equal(tt.congMark, lpp.GetCongMark(), tt.input)
			}
		}
	}
}

func TestEncodeLpHeaders(t *testing.T) {
	assert, require := makeAR(t)

	dpdktestenv.MakeMp("header", 63, 0,
		uint16(EncodeLpHeaders_GetHeadroom()+EncodeLpHeaders_GetTailroom()))

	tests := []struct {
		input  string
		output string
	}{
		{"6403 payload=5001A0", ""},
		{"6407 nack=FD032000(noreason) payload=5001A0", "6407 FD032000 5001"},
		{"640C nack=FD032005(FD03210196~noroute) payload=5001A0", "640C FD032005FD03210196 5001"},
		{"640E seq=5102A0A1 fragindex=520101 fragcount=530102 payload=5002D2D3",
			"6416 5108000000000000A0A1 52020001 53020002 5002"},
		{"640D pittoken=62089A414B412BC38EB2 payload=5001A0",
			"640D 62089A414B412BC38EB2 5001"},
	}
	for _, tt := range tests {
		inputPkt := packetFromHex(tt.input)
		defer inputPkt.Close()
		d := NewTlvDecoder(inputPkt)
		lpp, e := d.ReadLpPkt()
		require.NoError(e, tt.input)

		header := dpdktestenv.Alloc("header").AsPacket()
		defer header.Close()
		header.GetFirstSegment().SetHeadroom(EncodeLpHeaders_GetHeadroom())

		lpp.EncodeHeaders(header)
		assert.Equal(dpdktestenv.PacketBytesFromHex(tt.output), header.ReadAll(), tt.input)
	}
}
