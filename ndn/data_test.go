package ndn_test

import (
	"testing"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
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

func TestDataDecode(t *testing.T) {
	assert, _ := makeAR(t)

	var pkt ndn.Packet
	assert.NoError(tlv.Decode(bytesFromHex("060C name=0703080141 siginfo=16031B0100 sigvalue=1700"), &pkt))
	data := pkt.Data
	assert.NotNil(data)

	ndntestenv.NameEqual(assert, "/A", data)
	assert.Zero(data.ContentType)
	assert.Zero(data.Freshness)
	assert.Len(data.Content, 0)

	assert.NoError(tlv.Decode(bytesFromHex("0623 name=0706080142080130 "+
		"meta=140C contenttype=180103 freshness=19020104 finalblock=1A03080131 "+
		"content=1502C0C1 siginfo=16031B0100 unrecognized=F000 sigvalue=1700,"), &pkt))
	data = pkt.Data
	assert.NotNil(data)

	ndntestenv.NameEqual(assert, "/B/0", data)
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
