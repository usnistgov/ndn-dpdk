package ndn_test

import (
	"strings"
	"testing"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestNameParse(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		bad       bool
		err       error
		nComps    int
		hasDigest bool
	}{
		{input: "", nComps: 0},
		{input: "08FF DDDD", err: ndn.NdnError_Incomplete},
		{input: "FD800000", err: ndn.NdnError_BadNameComponentType},
		{input: "080141 080142 080100 0801FF 800141 0800 08012E", nComps: 7},
		{input: strings.Repeat("080141 ", 32) + "080142", nComps: 33},
		{input: "0120(DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A)",
			nComps: 1, hasDigest: true},
		{input: "0102 DDDD", err: ndn.NdnError_BadDigestComponentLength},
	}
	for _, tt := range tests {
		n, e := ndn.NewName(TlvBytesFromHex(tt.input))
		if tt.bad || tt.err != nil {
			assert.Error(e, tt.input)
			if tt.err != nil {
				assert.EqualError(e, tt.err.Error(), tt.input)
			}
		} else if assert.NoError(e, tt.input) {
			assert.Equal(tt.nComps, n.Len(), tt.input)
			assert.Equal(tt.hasDigest, n.HasDigestComp(), tt.input)
		}
	}
}

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
