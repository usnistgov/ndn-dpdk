package ndn

import (
	"testing"
	"time"
)

func TestInterest(t *testing.T) {
	assert, require := makeAR(t)

	checkNonce := func(nonce uint32) func() bool {
		return func() bool {
			return nonce == 0xCACBCCCD || nonce == 0xCDCCCBCA
		}
	}

	tests := []struct {
		input       string
		ok          bool
		name        string
		mustBeFresh bool
		lifetime    int
		fwHints     []string
	}{
		{"", false, "", false, 0, nil},
		{"0508 0700 0A04CACBCCCD", true, "/", false, 4000, nil},
		{"050E 0706080141080142 0A04CACBCCCD", true, "/A/B", false, 4000, nil},
		{"0515 0700 090B 0D0101 0E0101 110101 1200 0A04CACBCCCD", true, "/", true, 4000, nil},
		{"050B 0700 0A04CACBCCCD 0C01FF", true, "/", false, 255, nil},
		{"0514 0700 0A04CACBCCCD 1E0A 1F08 1E0100 0703080147", true,
			"/", false, 4000, []string{"/G"}},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.Truef(pkt.IsValid(), tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		interest, length, e := d.ReadInterest()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.EqualValues(pkt.Len(), length, tt.input)
				assert.Equal(tt.name, interest.GetName().String(), tt.input)
				assert.Equal(tt.mustBeFresh, interest.HasMustBeFresh(), tt.input)
				assert.Condition(checkNonce(interest.GetNonce()), tt.input)
				assert.EqualValues(tt.lifetime, interest.GetLifetime()/time.Millisecond, tt.input)

				fwHints := interest.GetFwHints()
				if assert.Len(fwHints, len(tt.fwHints), tt.input) {
					for i, fhName := range fwHints {
						assert.Equal(tt.fwHints[i], fhName.String(), tt.input, i)
					}
				}
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}
