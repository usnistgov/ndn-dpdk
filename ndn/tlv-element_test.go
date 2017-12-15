package ndn

import (
	"testing"
)

func TestTlvElement(t *testing.T) {
	assert, require := makeAR(t)

	const NOT_NNI uint64 = 0xB9C0CEA091E491F0
	tests := []struct {
		input       string
		ok          bool
		ttype       uint64
		length      int
		expectedNNI uint64
	}{
		{"", false, 0, 0, NOT_NNI},                              // empty
		{"01", false, 0, 0, NOT_NNI},                            // missing TLV-LENGTH
		{"01 01", false, 0, 0, NOT_NNI},                         // incomplete TLV-VALUE
		{"01 00", true, 0x01, 0x00, NOT_NNI},                    // zero TLV-LENGTH
		{"01 FF 01 00 00 00 00 00 00 00", false, 0, 0, NOT_NNI}, // TLV-LENGTH overflow
		{"01 01 01", true, 0x01, 0x01, 0x01},
		{"01 02 A0A1", true, 0x01, 0x02, 0xA0A1},
		{"01 03 A0A1A2", true, 0x01, 0x03, NOT_NNI},
		{"01 04 A0A1A2A3", true, 0x01, 0x04, 0xA0A1A2A3},
		{"01 05 A0A1A2A3A4", true, 0x01, 0x05, NOT_NNI},
		{"01 06 A0A1A2A3A4A5", true, 0x01, 0x06, NOT_NNI},
		{"01 07 A0A1A2A3A4A5A6", true, 0x01, 0x07, NOT_NNI},
		{"01 08 A0A1A2A3A4A5A6A7", true, 0x01, 0x08, 0xA0A1A2A3A4A5A6A7},
		{"01 09 A0A1A2A3A4A5A6A7A8", true, 0x01, 0x09, NOT_NNI},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.True(pkt.IsValid(), tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		ele, e := d.ReadTlvElement()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.Equal(pkt.Len(), ele.Len(), tt.input)
				assert.EqualValues(tt.ttype, ele.GetType(), tt.input)
				assert.Equal(tt.length, ele.GetLength(), tt.input)

				if nni, ok := ele.ReadNonNegativeInteger(); ok {
					assert.Equal(tt.expectedNNI, nni, tt.input)
				} else {
					assert.True(tt.expectedNNI == NOT_NNI)
				}
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}
