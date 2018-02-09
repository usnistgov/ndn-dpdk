package ndn_test

import (
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestNameEncode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input    string
		ok       bool
		outputTL string
		outputV  string
	}{
		{"ndn:/", true, "0700", ""},
		{"/", true, "0700", ""},
		{"/G", true, "0703", "080147"},
		{"/H/I", true, "0706", "080148 080149"},
		{"/.../..../.....", true, "0709", "0800 08012E 08022E2E"},
		{"/%00GH%ab%cD%EF", true, "0708", "0806004748ABCDEF"},
	}
	for _, tt := range tests {
		tlv, e1 := ndn.EncodeNameFromUri(tt.input)
		comps, e2 := ndn.EncodeNameComponentsFromUri(tt.input)
		if tt.ok {
			if assert.NoError(e1, tt.input) {
				expected := dpdktestenv.PacketBytesFromHex(tt.outputTL + tt.outputV)
				assert.EqualValues(expected, tlv, tt.input)
			}
			if assert.NoError(e2, tt.input) {
				expected := dpdktestenv.PacketBytesFromHex(tt.outputV)
				assert.EqualValues(expected, comps, tt.input)
			}
		} else {
			assert.Error(e1, tt.input)
			assert.Error(e2, tt.input)
		}
	}
}

func TestNameComponentEncodeFromNumber(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		tlvType ndn.TlvType
		v       interface{}
		output  string
	}{
		{ndn.TT_GenericNameComponent, uint8(0x5B), "08015B"},
		{ndn.TT_GenericNameComponent, uint16(0x7ED2), "08027ED2"},
		{ndn.TT_GenericNameComponent, uint32(0xD6793), "0804000D6793"},
		{ndn.TT_GenericNameComponent, uint64(0xEFF5DE886FF), "080800000EFF5DE886FF"},
	}
	for _, tt := range tests {
		encoded := ndn.EncodeNameComponentFromNumber(tt.tlvType, tt.v)
		expected := dpdktestenv.PacketBytesFromHex(tt.output)
		assert.EqualValues(expected, encoded, tt.output)
	}
}
