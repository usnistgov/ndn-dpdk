package ndn

import (
	"testing"
)

func TestReadTlvElement(t *testing.T) {
	assert, require := makeAR(t)

	tests := []struct {
		input  []byte
		ok     bool
		ttype  uint64
		length int
	}{
		{[]byte{0x01}, false, 0, 0},                  // empty
		{[]byte{0x01}, false, 0, 0},                  // missing TLV-LENGTH
		{[]byte{0x01, 0x01}, false, 0, 0},            // incomplete TLV-VALUE
		{[]byte{0x01, 0x00}, true, 0x01, 0x00},       // zero TLV-LENGTH
		{[]byte{0x01, 0x01, 0x01}, true, 0x01, 0x01}, // non-zero TLV-LENGTH
		{[]byte{0x01, 0xFF, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, false, 0, 0},
		// TLV-LENGTH overflow
	}
	for _, tt := range tests {
		pkt := packetFromBytes(tt.input)
		require.True(pkt.IsValid(), tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		ele, length, e := d.ReadTlvElement()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.Equal(len(tt.input), ele.Len(), tt.input)
				assert.EqualValues(tt.ttype, ele.GetType(), tt.input)
				assert.Equal(tt.length, ele.GetLength(), tt.input)
				assert.Equal(len(tt.input), length, tt.input)
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}
