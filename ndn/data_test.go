package ndn_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

func TestDataEncode(t *testing.T) {
	assert, _ := makeAR(t)

	var data ndn.Data
	data.Name = ndn.ParseName("/A")
	wire, e := tlv.Encode(data)
	assert.NoError(e)
	assert.Contains(string(wire), string(bytesFromHex("0703080141")))
	assert.Equal("/8=A", data.String())

	data = ndn.MakeData("/B", ndn.ContentType(3), 2500*time.Millisecond,
		[]byte{0xC0, 0xC1},
	)
	wire, e = tlv.Encode(data)
	assert.NoError(e)
	assert.Contains(string(wire),
		string(bytesFromHex("name=0703080142 meta=1407 contenttype=180103 freshness=190209C4 content=1502C0C1")))
}

func TestDataLpEncode(t *testing.T) {
	assert, _ := makeAR(t)

	var lph ndn.LpL3
	lph.PitToken = ndn.PitTokenFromUint(0xF0F1F2F3F4F5F6F7)
	lph.CongMark = 1
	interest := ndn.MakeInterest("/A", lph, ndn.NonceFromUint(0xC0C1C2C3), ndn.MustBeFreshFlag)
	data := ndn.MakeData(interest, bytesFromHex("content=C0C1"))

	wire, e := tlv.Encode(data.ToPacket())
	assert.NoError(e)
	assert.Contains(string(wire),
		string(bytesFromHex("pittoken=6208F7F6F5F4F3F2F1F0 congmark=FD03400101")))
	assert.Contains(string(wire),
		string(bytesFromHex("name=0703080141 meta=1403 freshness=190101 content=1502C0C1")))
}

func TestDataDecode(t *testing.T) {
	assert, _ := makeAR(t)

	var pkt ndn.Packet
	assert.NoError(tlv.Decode(bytesFromHex("060C name=0703080141 siginfo=16031B0100 sigvalue=1700"), &pkt))
	data := pkt.Data
	assert.NotNil(data)

	nameEqual(assert, "/A", data)
	assert.Zero(data.ContentType)
	assert.Zero(data.Freshness)
	assert.Len(data.Content, 0)

	assert.NoError(tlv.Decode(bytesFromHex("0623 name=0706080142080130 "+
		"meta=140C contenttype=180103 freshness=19020104 finalblock=1A03080131 "+
		"content=1502C0C1 siginfo=16031B0100 unrecognized=F000 sigvalue=1700,"), &pkt))
	data = pkt.Data
	assert.NotNil(data)

	nameEqual(assert, "/B/0", data)
	assert.EqualValues(3, data.ContentType)
	assert.Equal(260*time.Millisecond, data.Freshness)
	assert.Equal([]byte{0xC0, 0xC1}, data.Content)
}

func TestDataSatisfy(t *testing.T) {
	assert, _ := makeAR(t)

	interestExact := ndn.MakeInterest("/B")
	interestPrefix := ndn.MakeInterest("/B", ndn.CanBePrefixFlag)
	interestFresh := ndn.MakeInterest("/B", ndn.MustBeFreshFlag)

	tests := []struct {
		data        ndn.Data
		exactMatch  bool
		prefixMatch bool
		freshMatch  bool
	}{
		{ndn.MakeData("/A", time.Second),
			false, false, false},
		{ndn.MakeData("/2=B", time.Second),
			false, false, false},
		{ndn.MakeData("/B", time.Second),
			true, true, true},
		{ndn.MakeData("/B", time.Duration(0)),
			true, true, false},
		{ndn.MakeData("/B/0", time.Second),
			false, true, false},
		{ndn.MakeData("/", time.Second),
			false, false, false},
		{ndn.MakeData("/C", time.Second),
			false, false, false},
	}
	for i, tt := range tests {
		assert.Equal(tt.exactMatch, tt.data.CanSatisfy(interestExact), "%d", i)
		assert.Equal(tt.prefixMatch, tt.data.CanSatisfy(interestPrefix), "%d", i)
		assert.Equal(tt.freshMatch, tt.data.CanSatisfy(interestFresh), "%d", i)

		if tt.exactMatch {
			interestImplicit := ndn.MakeInterest(tt.data.FullName())
			assert.True(tt.data.CanSatisfy(interestImplicit))
		}
	}
}
