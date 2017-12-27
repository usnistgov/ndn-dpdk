package ndn

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestLpPkt(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input        string
		ok           bool
		hasPayload   bool
		isFragmented bool
		seqNo        uint64
		fragIndex    uint16
		fragCount    uint16
		nackReason   NackReason
		congMark     CongMark
	}{
		{"", false, false, false, 0, 0, 0, 0, 0},
		{"6406 payload=5004D0D1D2D3", true, true, false, 0, 0, 1, 0, 0},
		{"6402 unknown-critical=6300", false, false, false, 0, 0, 0, 0, 0},
		{"6404 unknown-critical=FD03BF00", false, false, false, 0, 0, 0, 0, 0},
		{"6404 unknown-ignored=FD03BC00", true, false, false, 0, 0, 1, 0, 0},
		{"6411 seq=5108A0A1A2A3A4A5A600 fragcount=530102 payload=5002D0D1", true,
			true, true, 0xA0A1A2A3A4A5A600, 0, 2, 0, 0},
		{"6414 seq=5108A0A1A2A3A4A5A601 fragindex=520101 fragcount=530102 payload=5002D2D3", true,
			true, true, 0xA0A1A2A3A4A5A601, 1, 2, 0, 0},
		{"6417 seq=5108A0A1A2A3A4A5A601 fragindex=520102 fragcount=530102 payload=5002D2D3", false,
			false, false, 0, 0, 0, 0, 0}, // FragIndex>=FragCount
		{"6404 nack=FD032000(noreason)", true,
			false, false, 0, 0, 1, NackReason_Unspecified, 0},
		{"6409 nack=FD032005(FD03210196~noroute)", true,
			false, false, 0, 0, 1, NackReason_NoRoute, 0},
		{"6405 congmark=FD03400104", true,
			false, false, 0, 0, 1, 0, 4},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		lpp, e := d.ReadLpPkt()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.Equal(tt.hasPayload, lpp.HasPayload(), tt.input)
				assert.Equal(tt.isFragmented, lpp.IsFragmented(), tt.input)
				seqNo, fragIndex, fragCount := lpp.GetFragFields()
				assert.Equal(tt.seqNo, seqNo, tt.input)
				assert.Equal(tt.fragIndex, fragIndex, tt.input)
				assert.Equal(tt.fragCount, fragCount, tt.input)
				assert.Equal(tt.nackReason, lpp.GetNackReason(), tt.input)
				assert.Equal(tt.congMark, lpp.GetCongMark(), tt.input)
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}

func TestEncodeLpHeaders(t *testing.T) {
	assert, require := makeAR(t)

	headerMp := dpdktestenv.MakeMp("header", 63, 0,
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
	}
	for _, tt := range tests {
		inputPkt := packetFromHex(tt.input)
		defer inputPkt.Close()
		d := NewTlvDecoder(inputPkt)
		lpp, e := d.ReadLpPkt()
		require.NoError(e, tt.input)

		headerMbuf, e := headerMp.Alloc()
		require.NoError(e)
		defer headerMbuf.Close()
		header := headerMbuf.AsPacket()
		header.GetFirstSegment().SetHeadroom(EncodeLpHeaders_GetHeadroom())

		lpp.EncodeHeaders(header)

		expected := dpdktestenv.PacketBytesFromHex(tt.output)
		assert.Equal(len(expected), header.Len(), tt.input)
		actual := make([]byte, len(expected))
		header.ReadTo(0, actual)
		assert.Equal(expected, actual, tt.input)
	}
}
