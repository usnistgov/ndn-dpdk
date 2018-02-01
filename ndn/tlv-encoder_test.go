package ndn

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
)

func TestTlvBytes(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		nElements int
	}{
		{"", 0},
		{"01", -1}, // missing TLV-LENGTH
		{"0100", 1},
		{"0101", -1}, // incomplete TLV-VALUE
		{"0101A0", 1},
		{"0202B0B1 01", -1}, // missing TLV-LENGTH
		{"0202B0B1 0100", 2},
		{"0202B0B1 0101", -1}, // incomplete TLV-VALUE
		{"0202B0B1 0101A0", 2},
	}
	for _, tt := range tests {
		tb := TlvBytes(dpdktestenv.PacketBytesFromHex(tt.input))
		assert.Equal(tt.nElements, tb.CountElements(), tt.input)
	}
}
