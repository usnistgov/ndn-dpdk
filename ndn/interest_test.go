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
		hopLimit    uint8
	}{
		{input: "", bad: true},
		{input: "0500", bad: true},
		{input: "0506 nonce=0A04A0A1A2A3", bad: true}, // Name missing
		{input: "0502 name=0700", bad: true},          // Name is empty
		{input: "0508 name=0706080141080142", name: "/A/B",
			lifetime: 4000, hopLimit: 0xFF},
		{input: "0528 name=0706080141080142 canbeprefix=2100 mustbefresh=1200 " +
			"fh=1E0A (del=1F08 pref=1E0100 name=0703080147) nonce=0A04A0A1A2A3 " +
			"lifetime=0C01FF hoplimit=220120 parameters=2302C0C1",
			name: "/A/B", canBePrefix: true, mustBeFresh: true, fhs: []string{"/G"},
			hasNonce: true, lifetime: 255, hopLimit: 0x20},
		{input: "050D name=0706080141080142 nonce=0A03A0A1A2", bad: true}, // Nonce wrong length
		{input: "0512 name=0706080141080142 lifetime=0C080000000100000000",
			bad: true}, // InterestLifetime too large
		{input: "050C name=0706080141080142 hoplimit=22020101",
			bad: true}, // HopLimit wrong length
		{input: "050B name=0706080141080142 hoplimit=220100", name: "/A/B",
			lifetime: 4000, hopLimit: 0x00},
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
			assert.Equal(-1, interest.GetActiveFhIndex(), tt.input)
			if fhs := interest.GetFhs(); assert.Len(fhs, len(tt.fhs), tt.input) {
				for i, fhName := range fhs {
					assert.Equal(tt.fhs[i], fhName.String(), "%s %i", tt.input, i)
				}
				if len(tt.fhs) > 0 {
					assert.Error(interest.SelectActiveFh(len(tt.fhs)), tt.input)
					assert.NoError(interest.SelectActiveFh(0), tt.input)
					assert.Equal(0, interest.GetActiveFhIndex(), tt.input)
					assert.NoError(interest.SelectActiveFh(-1), tt.input)
					assert.Equal(-1, interest.GetActiveFhIndex(), tt.input)
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

func TestInterestMatchesData(t *testing.T) {
	assert, require := makeAR(t)

	m := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	interestExact, e := ndn.MakeInterest(m, "/B")
	require.NoError(e)
	defer m.Close()

	m = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	interestPrefix, e := ndn.MakeInterest(m, "/B", ndn.CanBePrefixFlag)
	require.NoError(e)
	defer m.Close()

	m = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	interestFresh, e := ndn.MakeInterest(m, "/B", ndn.MustBeFreshFlag)
	require.NoError(e)
	defer m.Close()

	makeData := func(name string, args ...interface{}) *ndn.Data {
		m = dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
		data, e := ndn.MakeData(m, name, args...)
		require.NoError(e)
		return data
	}

	tests := []struct {
		data        *ndn.Data
		exactMatch  bool
		prefixMatch bool
		freshMatch  bool
	}{
		{makeData("/A", time.Second),
			false, false, false},
		{makeData("/2=B", time.Second),
			false, false, false},
		{makeData("/B", time.Second),
			true, true, true},
		{makeData("/B", time.Duration(0)),
			true, true, false},
		{makeData("/B/0", time.Second),
			false, true, false},
		{makeData("/", time.Second),
			false, false, false},
		{makeData("/C", time.Second),
			false, false, false},
	}
	for i, tt := range tests {
		assert.Equal(tt.exactMatch, interestExact.MatchesData(tt.data), "%d", i)
		assert.Equal(tt.prefixMatch, interestPrefix.MatchesData(tt.data), "%d", i)
		assert.Equal(tt.freshMatch, interestFresh.MatchesData(tt.data), "%d", i)
		tt.data.GetPacket().AsDpdkPacket().Close()
	}
}

func TestInterestModify(t *testing.T) {
	assert, _ := makeAR(t)

	const ins0 = " nonce=0A04A8A9AAAB lifetime=0C0400006D22 hoplimit=22017D"
	const pitToken0 = uint64(0xB7B6B5B4B3B2B1B0)
	const hopLimitDefault = ""

	tests := []struct {
		input  string
		output string
	}{
		{"6413 pittoken=6208B0B1B2B3B4B5B6B7 payload=5007 " +
			"0505 name=0703080141",
			"0514 name=0703080141" + ins0},
		{"050B name=0703080141 parameters=2304E0E1E2E3",
			"051A name=0703080141" + ins0 + " parameters=2304E0E1E2E3"},
		{"0507 name=0703080141 cbp=2100",
			"0516 name=0703080141 cbp=2100" + ins0},
		{"0507 name=0703080141 mbf=1200",
			"0516 name=0703080141 mbf=1200" + ins0},
		{"0511 name=0703080141 fh=1E0A1F081E01000703080147",
			"0520 name=0703080141 fh=1E0A1F081E01000703080147" + ins0},
		{"0518 name=0703080141 nonce=0A04A0A1A2A3 lifetime=0C02C0C1 hop=220180  parameters=2304E0E1E2E3",
			"051A name=0703080141" + ins0 + " parameters=2304E0E1E2E3"},
	}
	for i, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsDpdkPacket().Close()
		if e := pkt.ParseL2(); !assert.NoError(e, tt.input) {
			continue
		}
		if e := pkt.ParseL3(theMp); !assert.NoError(e, tt.input) {
			continue
		}
		interest := pkt.AsInterest()
		assert.Implements((*ndn.IL3Packet)(nil), interest)

		modified := interest.Modify(0xABAAA9A8, 27938*time.Millisecond, 125, theMp, theMp, theMp)
		if assert.NotNil(modified, tt.input) {
			npkt := modified.GetPacket()
			pkt := npkt.AsDpdkPacket()
			defer pkt.Close()

			assert.Equal(dpdktestenv.BytesFromHex(tt.output), pkt.ReadAll(), tt.input)
			if i == 0 {
				assert.Equal(pitToken0, npkt.GetLpL3().GetPitToken(), tt.input)
			}
		}
	}
}
