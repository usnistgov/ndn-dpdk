package ndn

import (
	"testing"
)

func TestReadVarNum(t *testing.T) {
	assert, _ := makeAR(t)

	decodeTests := []struct {
		input  []byte
		ok     bool
		output uint64
	}{
		{[]byte{}, false, 0},
		{[]byte{0x00}, true, 0x00},
		{[]byte{0xFC}, true, 0xFC},
		{[]byte{0xFD}, false, 0},
		{[]byte{0xFD, 0x00}, false, 0},
		{[]byte{0xFD, 0x01, 0x00}, true, 0x0100},
		{[]byte{0xFD, 0xFF, 0xFF}, true, 0xFFFF},
		{[]byte{0xFE, 0x00, 0x00, 0x00}, false, 0},
		{[]byte{0xFE, 0x01, 0x00, 0x00, 0x00}, true, 0x01000000},
		{[]byte{0xFE, 0xFF, 0xFF, 0xFF, 0xFF}, true, 0xFFFFFFFF},
		{[]byte{0xFF, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, false, 0},
		{[]byte{0xFF, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, true, 0x0100000000000000},
		{[]byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, true, 0xFFFFFFFFFFFFFFFF},
	}
	for _, tt := range decodeTests {
		pkt := packetFromBytes(tt.input)
		defer pkt.Close()
		assert.Truef(pkt.IsValid(), "%v", tt.input)
		d := NewTlvDecoder(pkt)
		v, length, e := d.ReadVarNum()
		if tt.ok {
			assert.NoErrorf(e, "%v", tt.input)
			assert.Equalf(tt.output, v, "%v", tt.input)
			assert.EqualValuesf(len(tt.input), length, "%v", tt.input)
		} else {
			assert.Error(e, "%v", tt.input)
		}
	}
}

func TestReadTlvElement(t *testing.T) {
	assert, _ := makeAR(t)

	decodeTests := []struct {
		input  []byte
		ok     bool
		ttype  uint64
		length uint
	}{
		{[]byte{0x01}, false, 0, 0},                  // empty
		{[]byte{0x01}, false, 0, 0},                  // missing TLV-LENGTH
		{[]byte{0x01, 0x01}, false, 0, 0},            // incomplete TLV-VALUE
		{[]byte{0x01, 0x00}, true, 0x01, 0x00},       // zero TLV-LENGTH
		{[]byte{0x01, 0x01, 0x01}, true, 0x01, 0x01}, // non-zero TLV-LENGTH
		{[]byte{0x01, 0xFF, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, false, 0, 0},
		// TLV-LENGTH overflow
	}
	for _, tt := range decodeTests {
		pkt := packetFromBytes(tt.input)
		defer pkt.Close()
		assert.Truef(pkt.IsValid(), "%v", tt.input)
		d := NewTlvDecoder(pkt)
		ele, length, e := d.ReadTlvElement()
		if tt.ok {
			assert.NoErrorf(e, "%v", tt.input)
			assert.EqualValuesf(len(tt.input), ele.Len(), "%v", tt.input)
			assert.Equalf(tt.ttype, ele.GetType(), "%v", tt.input)
			assert.Equalf(tt.length, ele.GetLength(), "%v", tt.input)
			assert.EqualValuesf(len(tt.input), length, "%v", tt.input)
		} else {
			assert.Error(e, "%v", tt.input)
		}
	}
}
