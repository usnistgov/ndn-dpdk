package ndn_test

import (
	"strings"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestNameComponent(t *testing.T) {
	assert, _ := makeAR(t)

	const aDigestHex = "DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A"

	tests := []struct {
		input string
		bad   bool
		t     uint64
		v     string
		str   string
	}{
		{input: "FD01", bad: true},         // incomplete TLV-TYPE
		{input: "08FD01", bad: true},       // incomplete TLV-LENGTH
		{input: "0802B0", bad: true},       // incomplete TLV-VALUE
		{input: "0802B0B1B2", bad: true},   // junk at end
		{input: "0001B0", bad: true},       // TLV-TYPE too small
		{input: "FE00010000", bad: true},   // TLV-TYPE too large
		{input: "0104B0B1B2B3", bad: true}, // ImplicitDigest wrong TLV-LENGTH

		{input: "0800", t: 0x08, v: "", str: "..."},
		{input: "08012E", t: 0x08, v: "2E", str: "...."},
		{input: "0200", t: 0x02, v: "", str: "2=..."},
		{input: "FC00", t: 0xFC, v: "", str: "252=..."},
		{input: "FDFFFF00", t: 0xFFFF, v: "", str: "65535=..."},
		{input: "96012E", t: 0x96, v: "2E", str: "150=...."},
		{input: "08052D2E5F7E41", t: 0x08, v: "2D2E5F7E41", str: "-._~A"},
		{input: "0804002081FF", t: 0x08, v: "002081FF", str: "%00%20%81%FF"},
		{input: "0120" + aDigestHex, t: 0x01, v: aDigestHex,
			str: "sha256digest=" + strings.ToLower(aDigestHex)},
	}

	for _, tt := range tests {
		comp := ndn.NameComponent(dpdktestenv.BytesFromHex(tt.input))
		if tt.bad {
			assert.False(comp.IsValid(), tt.input)
		} else if assert.True(comp.IsValid(), tt.input) {
			assert.Equal(ndn.TlvType(tt.t), comp.GetType(), tt.input)
			assert.Equal(ndn.TlvBytes(dpdktestenv.BytesFromHex(tt.v)), comp.GetValue(), tt.input)
			assert.Equal(tt.str, comp.String(), tt.input)

			parsed, e := ndn.ParseNameComponent(tt.str)
			if assert.NoError(e, tt.input) {
				assert.True(comp.Equal(parsed))
			}
		}
	}
}

func TestNameComponentFromNumber(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		tlvType ndn.TlvType
		v       interface{}
		output  string
	}{
		{ndn.TT_GenericNameComponent, uint8(0x5B), "08015B"},
		{ndn.TlvType(0x03), uint16(0x7ED2), "03027ED2"},
		{ndn.TlvType(0xFF), uint32(0xD6793), "FD00FF04000D6793"},
		{ndn.TlvType(0xFFFF), uint64(0xEFF5DE886FF), "FDFFFF0800000EFF5DE886FF"},
	}
	for _, tt := range tests {
		encoded := ndn.MakeNameComponentFromNumber(tt.tlvType, tt.v)
		expected := dpdktestenv.BytesFromHex(tt.output)
		assert.EqualValues(expected, encoded, tt.output)
	}
}
