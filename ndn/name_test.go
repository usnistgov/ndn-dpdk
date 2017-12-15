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
		{"", false, 0, false, ""},
		{"0700", true, 0, false, "/"},
		{"0714 080141 080142 080100 0801FF 800141 0800 08012E", true, 7, false,
			"/A/B/%00/%FF/128=A/.../...."},
		{"0722 0120(DC6D6840C6FAFB773D583CDBF465661C7B4B968E04ACD4D9015B1C4E53E59D6A)", true, 1, true,
			"/sha256digest=dc6d6840c6fafb773d583cdbf465661c7b4b968e04acd4d9015b1c4e53e59d6a"},
		{"0763 " + strings.Repeat("080141 ", 32) + "080142", true, 33, false,
			strings.Repeat("/A", 32) + "/B"},
		{"0200", false, 0, false, ""},           // bad TLV-TYPE
		{"0704 0102 DDDD", false, 0, false, ""}, // wrong digest length
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.True(pkt.IsValid(), tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		n, e := d.ReadName()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.Equal(tt.nComps, n.Len(), tt.input)
				assert.Equal(tt.hasDigest, n.HasDigest(), tt.input)
				assert.Equal(tt.str, n.String(), tt.input)
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}
