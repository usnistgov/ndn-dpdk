package ndn_test

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestReadVarNum(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input string
		bad   bool
		v     uint64
	}{
		{input: "", bad: true},
		{input: "00", v: 0x00},
		{input: "FC", v: 0xFC},
		{input: "FD", bad: true},
		{input: "FD 00", bad: true},
		{input: "FD 0100", v: 0x0100},
		{input: "FD FFFF", v: 0xFFFF},
		{input: "FE 000000", bad: true},
		{input: "FE 01000000", v: 0x01000000},
		{input: "FE FFFFFFFF", v: 0xFFFFFFFF},
		{input: "FF 00000000000000", bad: true},
		{input: "FF 0100000000000000", v: 0x0100000000000000},
		{input: "FF FFFFFFFFFFFFFFFF", v: 0xFFFFFFFFFFFFFFFF},
	}
	for _, tt := range tests {
		input := dpdktestenv.PacketBytesFromHex(tt.input)
		pkt := dpdktestenv.PacketFromBytes(input)
		defer pkt.Close()
		d := ndn.NewTlvDecodePos(pkt)

		v, e := d.ReadVarNum()
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			assert.Equal(tt.v, v, tt.input)
		}
	}
}
