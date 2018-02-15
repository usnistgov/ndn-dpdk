package ndn_test

import (
	"testing"
	"time"

	"ndn-dpdk/dpdk/dpdktestenv"
	"ndn-dpdk/ndn"
)

func TestDataDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input     string
		bad       bool
		name      string
		freshness int
	}{
		{input: "0600", bad: true},                        // missing Name
		{input: "0604 meta=1400 content=1500", bad: true}, // missing Name
		{input: "0605 name=0703080141", name: "/A"},
		{input: "0615 name=0703080142 meta=140C (180102 fp=190201FF 1A03080142) content=1500", name: "/B", freshness: 0x01FF},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsDpdkPacket().Close()
		e := pkt.ParseL3(theMp)
		if tt.bad {
			assert.Error(e, tt.input)
		} else if assert.NoError(e, tt.input) {
			if !assert.Equal(ndn.L3PktType_Data, pkt.GetL3Type(), tt.input) {
				continue
			}
			data := pkt.AsData()
			assert.Equal(tt.name, data.GetName().String(), tt.input)
			assert.EqualValues(tt.freshness, data.GetFreshnessPeriod()/time.Millisecond, tt.input)
		}
	}
}

func TestDataEncode(t *testing.T) {
	assert, require := makeAR(t)

	name, e := ndn.NewName(TlvBytesFromHex("080141 080142"))
	require.NoError(e)

	payloadMbuf := dpdktestenv.PacketFromHex("C0C1C2C3C4C5C6C7")
	// note: payloadMbuf will be leaked if there's a fatal error below

	m1 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	m2 := dpdktestenv.Alloc(dpdktestenv.MPID_DIRECT)
	encoded := ndn.EncodeData(name, payloadMbuf, m1, m2)

	pkt := ndn.PacketFromDpdk(encoded)
	e = pkt.ParseL3(theMp)
	require.NoError(e)
	data := pkt.AsData()

	assert.Equal(2, data.GetName().Len())
	// assert.Equal("/A/B", data.GetName().String())
}
