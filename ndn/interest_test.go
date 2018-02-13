package ndn_test

import (
	"encoding/binary"
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
			"lifetime=0C01FF hoplimit=250120 parameters=2302C0C1",
			name: "/A/B", canBePrefix: true, mustBeFresh: true, fhs: []string{"/G"},
			hasNonce: true, lifetime: 255, hopLimit: 31}, // HopLimit decremented
		{input: "050D name=0706080141080142 nonce=0A03A0A1A2", bad: true}, // Nonce wrong length
		{input: "0512 name=0706080141080142 lifetime=0C080000000100000000",
			bad: true}, // InterestLifetime too large
		{input: "050C name=0706080141080142 hoplimit=25020101",
			bad: true}, // HopLimit wrong length
		{input: "050B name=0706080141080142 hoplimit=250100", name: "/A/B",
			lifetime: 4000, hopLimit: ndn.HOP_LIMIT_ZERO},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsDpdkPacket().Close()
		e := pkt.ParseL3(parseMps)
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			if !assert.Equal(ndn.L3PktType_Interest, pkt.GetL3Type(), tt.input) {
				continue
			}
			interest := pkt.AsInterest()
			// assert.Equal(tt.name, interest.GetName().String(), tt.input)
			assert.Equal(tt.canBePrefix, interest.HasCanBePrefix(), tt.input)
			assert.Equal(tt.mustBeFresh, interest.HasMustBeFresh(), tt.input)
			if fhs := interest.GetFhs(); assert.Len(fhs, len(tt.fhs), tt.input) {
				// for i, fhName := range fhs {
				// 	assert.Equal(tt.fhs[i], fhName.String(), "%s %i", tt.input, i)
				// }
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

func checkEncodeInterest(t *testing.T, tpl *ndn.InterestTemplate,
	expectedHex string, nonceOffset int) {
	assert, _ := makeAR(t)

	expected := dpdktestenv.PacketBytesFromHex(expectedHex)

	pkt := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT).AsPacket()
	tpl.EncodeTo(pkt)
	assert.Equal(len(expected), pkt.Len())

	actual := make([]byte, len(expected))
	pkt.ReadTo(0, actual)
	assert.NotEqual(binary.LittleEndian.Uint32(actual[nonceOffset:nonceOffset+4]),
		uint32(0xCCCCCCCC))
	binary.LittleEndian.PutUint32(actual[nonceOffset:nonceOffset+4], 0xCCCCCCCC)
	assert.Equal(expected, actual)
}

func TestEncodeInterest0(t *testing.T) {
	assert, _ := makeAR(t)

	tpl := ndn.NewInterestTemplate()
	e := tpl.SetNamePrefixFromUri("/")
	assert.NoError(e)
	tpl.SetMustBeFresh(false)
	assert.False(tpl.GetMustBeFresh())
	assert.Equal(4000*time.Millisecond, tpl.GetInterestLifetime())

	checkEncodeInterest(t, tpl,
		"050E 0700 0A04CCCCCCCC 0C0400000FA0", 6)
}

func TestEncodeInterest1(t *testing.T) {
	assert, _ := makeAR(t)

	tpl := ndn.NewInterestTemplate()
	e := tpl.SetNamePrefixFromUri("/A/B")
	assert.NoError(e)
	tpl.NameSuffix = ndn.EncodeNameComponentFromNumber(ndn.TT_GenericNameComponent,
		uint32(0x737F2FBD))
	tpl.SetMustBeFresh(true)
	assert.True(tpl.GetMustBeFresh())
	tpl.SetInterestLifetime(9000 * time.Millisecond)
	assert.Equal(9000*time.Millisecond, tpl.GetInterestLifetime())

	checkEncodeInterest(t, tpl,
		"051E 070C0801410801420804737F2FBD 09021200 0A04CCCCCCCC 0C0400002328", 22)
}
