package ndn_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestInterestEncode(t *testing.T) {
	assert, _ := makeAR(t)

	var interest ndn.Interest
	interest.Name = ndn.ParseName("/A")
	wire, e := tlv.Encode(interest)
	assert.NoError(e)
	assert.Len(wire, 13)
	assert.Equal(bytesFromHex("050B 0703080141 0A04"), wire[:9])
	assert.Equal("/8=A", interest.String())

	interest = ndn.MakeInterest("/B", ndn.CanBePrefixFlag, ndn.MustBeFreshFlag,
		ndn.MakeFHDelegation(33, "/FH"), ndn.NonceFromUint(0x85AC8579),
		8198*time.Millisecond, ndn.HopLimit(5),
	)
	wire, e = tlv.Encode(interest)
	assert.NoError(e)
	assert.Equal(bytesFromHex("0523 name=0703080142 cbp=2100 mbf=1200 "+
		"fh=1E0B1F091E0121070408024648 nonce=0A047985AC85 lifetime=0C022006 hoplimit=220105"), wire)
	assert.Equal("/8=B[P][F]", interest.String())
}

func TestInterestLpEncode(t *testing.T) {
	assert, _ := makeAR(t)

	var lph ndn.LpHeader
	lph.PitToken = ndn.PitTokenFromUint(0xF0F1F2F3F4F5F6F7)
	interest := ndn.MakeInterest("/A", lph, ndn.NonceFromUint(0xC0C1C2C3))

	wire, e := tlv.Encode(interest.Packet)
	assert.NoError(e)
	assert.Equal(bytesFromHex("6419 pittoken=6208F7F6F5F4F3F2F1F0 payload=500D "+
		"interest=050B 0703080141 0A04C3C2C1C0"), wire)
}

func TestInterestDecode(t *testing.T) {
	assert, _ := makeAR(t)

	var pkt ndn.Packet
	assert.NoError(tlv.Decode(bytesFromHex("0505 0703080141"), &pkt))
	interest := pkt.Interest
	assert.NotNil(interest)

	nameEqual(assert, "/A", interest)
	assert.False(interest.CanBePrefix)
	assert.False(interest.MustBeFresh)

	assert.NoError(tlv.Decode(bytesFromHex("0523 name=0703080141 cbp=2100 mbf=1200 "+
		"fh=1E0B1F091E0121070408024648 nonce=0A04A0A1A2A3 lifetime=0C0276A1 hoplimit=2201DC"), &pkt))
	interest = pkt.Interest
	assert.NotNil(interest)

	nameEqual(assert, "/A", interest)
	assert.True(interest.CanBePrefix)
	assert.True(interest.MustBeFresh)
	assert.Len(interest.ForwardingHint, 1)
	assert.Equal(ndn.Nonce{0xA0, 0xA1, 0xA2, 0xA3}, interest.Nonce)
	assert.Equal(30369*time.Millisecond, interest.Lifetime)
	assert.EqualValues(220, interest.HopLimit)
}
