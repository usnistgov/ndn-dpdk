package tlv_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestElement(t *testing.T) {
	assert, _ := makeAR(t)

	const NOT_NNI uint64 = 0xB9C0CEA091E491F0
	tests := []struct {
		input string
		bad   bool
		t     uint32
		v     string
		nni   uint64
	}{
		{input: "", bad: true},                         // empty
		{input: "01", bad: true},                       // missing TLV-LENGTH
		{input: "01 01", bad: true},                    // incomplete TLV-VALUE
		{input: "01 FD00", bad: true},                  // incomplete TLV-LENGTH
		{input: "01 FF0000000100000000 A0", bad: true}, // TLV-LENGTH overflow
		{input: "01 04 A0A1", bad: true},               // incomplete TLV-VALUE
		{input: "01 00", t: 0x01, v: "", nni: NOT_NNI}, // zero TLV-LENGTH
		{input: "01 01 01", t: 0x01, v: "01", nni: 0x01},
		{input: "01 02 A0A1", t: 0x01, v: "A0A1", nni: 0xA0A1},
		{input: "01 03 A0A1A2", t: 0x01, v: "A0A1A2", nni: NOT_NNI},
		{input: "01 04 A0A1A2A3", t: 0x01, v: "A0A1A2A3", nni: 0xA0A1A2A3},
		{input: "01 05 A0A1A2A3A4", t: 0x01, v: "A0A1A2A3A4", nni: NOT_NNI},
		{input: "01 06 A0A1A2A3A4A5", t: 0x01, v: "A0A1A2A3A4A5", nni: NOT_NNI},
		{input: "01 07 A0A1A2A3A4A5A6", t: 0x01, v: "A0A1A2A3A4A5A6", nni: NOT_NNI},
		{input: "01 08 A0A1A2A3A4A5A6A7", t: 0x01, v: "A0A1A2A3A4A5A6A7", nni: 0xA0A1A2A3A4A5A6A7},
		{input: "01 09 A0A1A2A3A4A5A6A7A8", t: 0x01, v: "A0A1A2A3A4A5A6A7A8", nni: NOT_NNI},
	}
	for _, tt := range tests {
		input := bytesFromHex(tt.input)
		var element tlv.Element
		rest, e := element.UnmarshalTlv(input)

		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			assert.Equal(len(input), element.Size(), tt.input)
			assert.Len(rest, 0, tt.input)

			assert.Equal(tt.t, element.Type, tt.input)
			value := bytesFromHex(tt.v)
			assert.Equal(len(value), element.Length(), tt.input)
			bytesEqual(assert, value, element.Value, tt.input)

			var nni tlv.NNI
			if e := nni.UnmarshalBinary(value); e == nil {
				assert.EqualValues(tt.nni, nni, tt.input)

				nniV, _ := nni.MarshalBinary()
				assert.Equal(value, nniV)
			} else {
				assert.True(tt.nni == NOT_NNI, tt.input)
			}
		}
	}
}
