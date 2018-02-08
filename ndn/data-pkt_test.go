package ndn_test

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestData(t *testing.T) {
	assert, _ := makeAR(t)

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
		defer pkt.Close()
		d := ndn.NewTlvDecoder(pkt)

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

func TestEncodeData(t *testing.T) {
	assert, require := makeAR(t)

	nameMbuf := packetFromHex("0706 080141 080142")
	defer nameMbuf.Close()
	d := ndn.NewTlvDecoder(nameMbuf)
	name, e := d.ReadName()
	require.NoError(e)

	payloadMbuf := packetFromHex("C0C1C2C3C4C5C6C7")
	// note: payloadMbuf will be leaked if there's a fatal error below

	m1 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	m2 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)

	encoded := ndn.EncodeData(&name, payloadMbuf, m1, m2)
	d = ndn.NewTlvDecoder(encoded)
	data, e := d.ReadData()
	require.NoError(e)

	assert.Equal("/A/B", data.GetName().String())
}
