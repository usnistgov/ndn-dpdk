package ndn_test

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk"
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

func TestInterestModify(t *testing.T) {
	assert, _ := makeAR(t)

	mod0 := ndn.InterestMod{
		Nonce:    0xABAAA9A8,
		Lifetime: 27938 * time.Millisecond,
		HopLimit: ndn.HOP_LIMIT_OMITTED,
	}
	ins0 := " nonce=0A04A8A9AAAB lifetime=0C0400006D22"
	mod1 := mod0
	mod1.HopLimit = 41
	ins1 := ins0 + " hop=220129"

	tests := []struct {
		input string
		out0  string
		out1  string
	}{
		{"0505 name=0703080141",
			"0511 name=0703080141" + ins0,
			"0514 name=0703080141" + ins1},
		{"050B name=0703080141 parameters=2304E0E1E2E3",
			"0517 name=0703080141" + ins0 + " parameters=2304E0E1E2E3",
			"051A name=0703080141" + ins1 + " parameters=2304E0E1E2E3"},
		{"0507 name=0703080141 cbp=2100",
			"0513 name=0703080141 cbp=2100" + ins0,
			"0516 name=0703080141 cbp=2100" + ins1},
		{"0507 name=0703080141 mbf=1200",
			"0513 name=0703080141 mbf=1200" + ins0,
			"0516 name=0703080141 mbf=1200" + ins1},
		{"0511 name=0703080141 fh=1E0A1F081E01000703080147",
			"051D name=0703080141 fh=1E0A1F081E01000703080147" + ins0,
			"0520 name=0703080141 fh=1E0A1F081E01000703080147" + ins1},
		{"0518 name=0703080141 nonce=0A04A0A1A2A3 lifetime=0C02C0C1 hop=220180  parameters=2304E0E1E2E3",
			"0517 name=0703080141" + ins0 + " parameters=2304E0E1E2E3",
			"051A name=0703080141" + ins1 + " parameters=2304E0E1E2E3"},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsDpdkPacket().Close()
		e := pkt.ParseL3(theMp)
		if !assert.NoError(e, tt.input) {
			continue
		}
		interest := pkt.AsInterest()

		var mbufs [4]dpdk.Mbuf
		dpdktestenv.AllocBulk(dpdktestenv.MPID_DIRECT, mbufs[:])

		out0 := interest.Modify(mod0, mbufs[0], mbufs[1], theMp)
		if assert.NotNil(out0, tt.input) {
			pkt0 := out0.GetPacket().AsDpdkPacket()
			assert.Equal(dpdktestenv.BytesFromHex(tt.out0),
				pkt0.ReadAll(), tt.input)
		}

		out1 := interest.Modify(mod1, mbufs[2], mbufs[3], theMp)
		if assert.NotNil(out1, tt.input) {
			pkt1 := out1.GetPacket().AsDpdkPacket()
			assert.Equal(dpdktestenv.BytesFromHex(tt.out1),
				pkt1.ReadAll(), tt.input)
		}

		// TODO verify that LpL3 is copied
	}
}
