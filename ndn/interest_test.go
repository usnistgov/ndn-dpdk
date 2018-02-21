package ndn_test

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestInterestDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input       string
		bad         bool
		name        string
		canBePrefix bool
		mustBeFresh bool
		fhs         []string
		hasNonce    bool
		lifetime    int
		hopLimit    ndn.HopLimit
	}{
		{input: "", bad: true},
		{input: "0500", bad: true},
		{input: "0506 nonce=0A04A0A1A2A3", bad: true}, // Name missing
		{input: "0502 name=0700", bad: true},          // Name is empty
		{input: "0508 name=0706080141080142", name: "/A/B",
			lifetime: 4000, hopLimit: ndn.HOP_LIMIT_OMITTED},
		{input: "0528 name=0706080141080142 canbeprefix=2100 mustbefresh=1200 " +
			"fh=1E0A (del=1F08 pref=1E0100 name=0703080147) nonce=0A04A0A1A2A3 " +
			"lifetime=0C01FF hoplimit=220120 parameters=2302C0C1",
			name: "/A/B", canBePrefix: true, mustBeFresh: true, fhs: []string{"/G"},
			hasNonce: true, lifetime: 255, hopLimit: 31}, // HopLimit decremented
		{input: "050D name=0706080141080142 nonce=0A03A0A1A2", bad: true}, // Nonce wrong length
		{input: "0512 name=0706080141080142 lifetime=0C080000000100000000",
			bad: true}, // InterestLifetime too large
		{input: "050C name=0706080141080142 hoplimit=22020101",
			bad: true}, // HopLimit wrong length
		{input: "050B name=0706080141080142 hoplimit=220100", name: "/A/B",
			lifetime: 4000, hopLimit: ndn.HOP_LIMIT_ZERO},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsDpdkPacket().Close()
		e := pkt.ParseL3(theMp)
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			if !assert.Equal(ndn.L3PktType_Interest, pkt.GetL3Type(), tt.input) {
				continue
			}
			interest := pkt.AsInterest()
			assert.Equal(tt.name, interest.GetName().String(), tt.input)
			assert.Equal(tt.canBePrefix, interest.HasCanBePrefix(), tt.input)
			assert.Equal(tt.mustBeFresh, interest.HasMustBeFresh(), tt.input)
			assert.Equal(-1, interest.GetFhIndex(), tt.input)
			if fhs := interest.GetFhs(); assert.Len(fhs, len(tt.fhs), tt.input) {
				for i, fhName := range fhs {
					assert.Equal(tt.fhs[i], fhName.String(), "%s %i", tt.input, i)
				}
				if len(tt.fhs) > 0 {
					assert.Error(interest.SetFhIndex(len(tt.fhs)), tt.input)
					assert.NoError(interest.SetFhIndex(0), tt.input)
					assert.Equal(0, interest.GetFhIndex(), tt.input)
				}
			}
			if tt.hasNonce {
				assert.Equal(uint32(0xA3A2A1A0), interest.GetNonce(), tt.input)
			} else {
				assert.Zero(interest.GetNonce(), tt.input)
			}
			assert.EqualValues(tt.lifetime, interest.GetLifetime()/time.Millisecond, tt.input)
			assert.Equal(tt.hopLimit, interest.GetHopLimit(), tt.input)
		}
	}
}

func TestInterestEncode(t *testing.T) {
	assert, require := makeAR(t)

	tpl := ndn.NewInterestTemplate()
	prefix, e := ndn.ParseName("/A/B")
	require.NoError(e)
	tpl.SetNamePrefix(prefix)

	pkt := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	tpl.Encode(pkt, nil, nil)
	encoded := pkt.ReadAll()
	require.Len(encoded, 16)
	assert.Equal(dpdktestenv.BytesFromHex("050E name=0706080141080142 nonce=0A04"),
		encoded[:12])

	tpl.SetCanBePrefix(true)
	tpl.SetMustBeFresh(true)
	tpl.SetInterestLifetime(9000 * time.Millisecond)
	tpl.SetHopLimit(125)

	suffix, e := ndn.ParseName("/C/D")
	require.NoError(e)

	pkt = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	defer pkt.Close()
	tpl.Encode(pkt, suffix, nil)
	encoded = pkt.ReadAll()
	require.Len(encoded, 35)
	assert.Equal(dpdktestenv.BytesFromHex("0521 name=070C080141080142080143080144 "+
		"canbeprefix=2100 mustbefresh=1200 nonce=0A04"), encoded[:22])
	assert.Equal(dpdktestenv.BytesFromHex("lifetime=0C0400002328 hoplimit=22017D"),
		encoded[26:])
}
