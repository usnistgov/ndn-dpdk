package ndn

import (
	"testing"
)

func TestLpPkt(t *testing.T) {
	assert, require := makeAR(t)

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
		{"6404 nack=FD032000(no reason)", true,
			false, false, 0, 0, 1, NackReason_Unspecified, 0},
		{"6409 nack=FD032005(FD03210196~noroute)", true,
			false, false, 0, 0, 1, NackReason_NoRoute, 0},
		{"6405 congmark=FD03400104", true,
			false, false, 0, 0, 1, 0, 4},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.True(pkt.IsValid(), tt.input)
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
