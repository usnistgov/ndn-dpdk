package ndn

import (
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	assert, require := makeAR(t)

	tests := []struct {
		input     string
		ok        bool
		nComps    int
		hasDigest bool
		str       string
	}{
		{"07 00", true, 0, false, "/"},
		{"07 14 08 01 41 08 01 42 08 01 00 08 01 FF 80 01 41 08 00 08 01 2E", true, 7, false,
			"/A/B/%00/%FF/128=A/.../...."},
		{"07 22 01 20 DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A", true, 1, true,
			"/sha256digest=dc6d6840c6fafb773d583cdbf465661c7b4b968e04acd4d9015b1c4e53e59d6a"},
		{"07 63 " + strings.Repeat("08 01 41 ", 32) + "08 01 42", true, 33, false,
			strings.Repeat("/A", 32) + "/B"},
		{"02 00", false, 0, false, ""},            // bad TLV-TYPE
		{"07 04 01 02 DDDD", false, 0, false, ""}, // wrong digest length
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.Truef(pkt.IsValid(), "%v", tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		n, length, e := d.ReadName()
		if tt.ok {
			assert.NoErrorf(e, "%v", tt.input)
			assert.EqualValuesf(pkt.Len(), length, "%v", tt.input)
			assert.Equalf(tt.nComps, n.Len(), "%v", tt.input)
			assert.Equalf(tt.hasDigest, n.HasDigest(), "%v", tt.input)
			assert.Equalf(tt.str, n.String(), "%v", tt.input)
		} else {
			assert.Error(e, "%v", tt.input)
		}
	}
}
