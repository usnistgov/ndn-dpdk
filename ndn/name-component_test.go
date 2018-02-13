package ndn_test

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestNameComponentFromNumber(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		tlvType ndn.TlvType
		v       interface{}
		output  string
	}{
		{ndn.TT_GenericNameComponent, uint8(0x5B), "08015B"},
		{ndn.TlvType(0x02), uint16(0x7ED2), "02027ED2"},
		{ndn.TlvType(0xFF), uint32(0xD6793), "FD00FF04000D6793"},
		{ndn.TlvType(0xFFFF), uint64(0xEFF5DE886FF), "FDFFFF0800000EFF5DE886FF"},
	}
	for _, tt := range tests {
		encoded := ndn.MakeNameComponentFromNumber(tt.tlvType, tt.v)
		expected := dpdktestenv.PacketBytesFromHex(tt.output)
		assert.EqualValues(expected, encoded, tt.output)
	}
}
