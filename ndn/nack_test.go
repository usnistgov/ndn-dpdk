package ndn_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestNackLpEncode(t *testing.T) {
	assert, _ := makeAR(t)

	var lph ndn.LpL3
	lph.PitToken = bytesFromHex("B0B1B2")
	interest := ndn.MakeInterest("/A", lph, ndn.NonceFromUint(0xC0C1C2C3))

	nackNoReason := ndn.MakeNack(interest)
	wire, e := tlv.EncodeFrom(nackNoReason.ToPacket())
	assert.NoError(e)
	assert.Equal(bytesFromHex("6418 pittoken=6203B0B1B2 nack=FD032000 payload=500D "+
		"interest=050B 0703080141 0A04C0C1C2C3"), wire)

	nackNoRoute := ndn.MakeNack(interest)
	nackNoRoute.Reason = an.NackNoRoute
	wire, e = tlv.EncodeFrom(nackNoRoute.ToPacket())
	assert.NoError(e)
	assert.Equal(bytesFromHex("641D pittoken=6203B0B1B2 nack=FD032005FD03210196 payload=500D "+
		"interest=050B 0703080141 0A04C0C1C2C3"), wire)

	nackDuplicate := ndn.MakeNack(&interest, an.NackDuplicate)
	wire, e = tlv.EncodeFrom(nackDuplicate.ToPacket())
	assert.NoError(e)
	assert.Equal(bytesFromHex("641D pittoken=6203B0B1B2 nack=FD032005FD03210164 payload=500D "+
		"interest=050B 0703080141 0A04C0C1C2C3"), wire)
}

func TestNackDecode(t *testing.T) {
	assert, _ := makeAR(t)

	var pkt ndn.Packet
	assert.NoError(tlv.Decode(bytesFromHex("6418 pittoken=6203B0B1B2 nack=FD032000 payload=500D "+
		"interest=050B 0703080141 0A04A0A1A2A3"), &pkt))
	nackNoReason := pkt.Nack
	assert.NotNil(nackNoReason)

	assert.EqualValues(an.NackUnspecified, nackNoReason.Reason)
	nameEqual(assert, "/A", nackNoReason)
	assert.Equal(ndn.Nonce{0xA0, 0xA1, 0xA2, 0xA3}, nackNoReason.Interest.Nonce)
	assert.Equal("/8=A~unspecified", nackNoReason.String())

	assert.NoError(tlv.Decode(bytesFromHex("641D pittoken=6203B0B1B2 nack=FD032005FD03210196 payload=500D "+
		"interest=050B 0703080141 0A04A0A1A2A3"), &pkt))
	nackNoRoute := pkt.Nack
	assert.NotNil(nackNoRoute)

	assert.EqualValues(an.NackNoRoute, nackNoRoute.Reason)
	nameEqual(assert, "/A", nackNoRoute)
	assert.Equal(ndn.Nonce{0xA0, 0xA1, 0xA2, 0xA3}, nackNoRoute.Interest.Nonce)
	assert.Equal("/8=A~no-route", nackNoRoute.String())
}
