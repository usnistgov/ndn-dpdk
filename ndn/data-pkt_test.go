package ndn

import (
	"testing"
	"time"
)

func TestData(t *testing.T) {
	assert, require := makeAR(t)

	tests := []struct {
		input     string
		ok        bool
		name      string
		freshness int
	}{
		{"", false, "", 0},
		{"0609 name=0703080141 meta=1400 content=1500", true, "/A", 0},
		{"0615 name=0703080142 meta=140C (180102 fp=190201FF 1A03080142) content=1500", true, "/B", 0x01FF},
		{"0604 meta=1400 content=1500", false, "", 0}, // missing Name
		{"0604 name=0700 meta=1400", false, "", 0},    // missing MetaInfo
		{"0604 name=0700 content=1500", false, "", 0}, // missing Content
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		require.Truef(pkt.IsValid(), tt.input)
		defer pkt.Close()
		d := NewTlvDecoder(pkt)

		data, e := d.ReadData()
		if tt.ok {
			if assert.NoError(e, tt.input) {
				assert.Equal(tt.name, data.GetName().String(), tt.input)
				assert.EqualValues(tt.freshness, data.GetFreshnessPeriod()/time.Millisecond, tt.input)
			}
		} else {
			assert.Error(e, tt.input)
		}
	}
}
