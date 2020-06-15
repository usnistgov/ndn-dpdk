package ndn_test

import (
	"testing"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/ndntestenv"
)

func TestNackDecode(t *testing.T) {
	assert, _ := makeAR(t)

	tests := []struct {
		input  string
		reason ndn.NackReason
	}{
		{input: "6413 nack=FD032000(noreason) payload=500D 050B 0703080141 0A04A0A1A2A3",
			reason: ndn.NackReason_Unspecified},
		{input: "6418 nack=FD032005(FD03210196~noroute) payload=500D 050B 0703080141 0A04A0A1A2A3",
			reason: ndn.NackReason_NoRoute},
	}
	for _, tt := range tests {
		pkt := packetFromHex(tt.input)
		defer pkt.AsMbuf().Close()
		if !assert.NoError(pkt.ParseL2(), tt.input) {
			continue
		}
		if !assert.NoError(pkt.ParseL3(ndntestenv.Name.Pool()), tt.input) {
			continue
		}
		if !assert.Equal(ndn.L3PktType_Nack, pkt.GetL3Type(), tt.input) {
			continue
		}
		nack := pkt.AsNack()
		assert.Implements((*ndn.IL3Packet)(nil), nack)
		assert.Equal(tt.reason, nack.GetReason(), tt.input)
		interest := nack.GetInterest()
		ndntestenv.NameEqual(assert, "/A", interest, tt.input)
		assert.Equal(uint32(0xA3A2A1A0), interest.GetNonce(), tt.input)
	}
}
