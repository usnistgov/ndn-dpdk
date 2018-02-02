package ndn

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestTlvBytes(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input        string
		nElements    int
		elementSizes []int
	}{
		{"", 0, []int{}},
		{"01", -1, nil}, // missing TLV-LENGTH
		{"0100", 1, []int{2}},
		{"0101", -1, nil}, // incomplete TLV-VALUE
		{"0101A0", 1, []int{3}},
		{"0202B0B1 01", -1, nil}, // missing TLV-LENGTH
		{"0202B0B1 0100", 2, []int{4, 2}},
		{"0202B0B1 0101", -1, nil}, // incomplete TLV-VALUE
		{"0202B0B1 0101A0", 2, []int{4, 3}},
	}
	for _, tt := range tests {
		tb := TlvBytes(dpdktestenv.PacketBytesFromHex(tt.input))
		assert.Equal(tt.nElements, tb.CountElements(), tt.input)
		if elements := tb.SplitElements(); tt.nElements == -1 {
			assert.Nil(elements, tt.input)
		} else if assert.NotNil(elements, tt.input) {
			assert.Len(elements, len(tt.elementSizes), tt.input)
			for i, element := range elements {
				assert.Len(element, tt.elementSizes[i], "%s [%d]", tt.input, i)
			}
		}
	}
}
