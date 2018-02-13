package ndn_test

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestTlvBytes(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input        string
		elementSizes []int
	}{
		{"", []int{}},
		{"01", nil}, // missing TLV-LENGTH
		{"0100", []int{2}},
		{"0101", nil}, // incomplete TLV-VALUE
		{"0101A0", []int{3}},
		{"0202B0B1 01", nil}, // missing TLV-LENGTH
		{"0202B0B1 0100", []int{4, 2}},
		{"0202B0B1 0101", nil}, // incomplete TLV-VALUE
		{"0202B0B1 0101A0", []int{4, 3}},
	}
	for _, tt := range tests {
		tb := ndn.TlvBytes(dpdktestenv.PacketBytesFromHex(tt.input))
		nElements := tb.CountElements()
		elements := tb.SplitElements()
		if tt.elementSizes == nil {
			assert.Equal(-1, nElements, tt.input)
			assert.Nil(elements, tt.input)
		} else if assert.NotNil(elements, tt.input) {
			assert.Equal(len(tt.elementSizes), nElements, tt.input)
			assert.Len(elements, len(tt.elementSizes), tt.input)
			accum := 0
			for i, element := range elements {
				size := tt.elementSizes[i]
				assert.Len(element, size, "%s [%d]", tt.input, i)
				assert.True(element.Equal(tb[accum : accum+size]))
				accum += size
			}
		}
	}
}
