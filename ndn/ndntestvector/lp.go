package ndntestvector

import "github.com/usnistgov/ndn-dpdk/ndn/an"

const (
	bareInterest     = "0505 0703080141"
	payloadInterest  = "5007 " + bareInterest
	payloadInterestL = 7
	payloadFragment  = "5004 D0D1D2D3"
	payloadFragmentL = 4
)

// LpDecodeTests contains test vectors for NDNLPv2 decoder.
var LpDecodeTests = []struct {
	Input      string
	Bad        bool
	SeqNum     uint64
	FragIndex  uint16
	FragCount  uint16
	PitToken   uint64
	NackReason uint8
	CongMark   uint8
	PayloadL   int
}{
	{Input: "", Bad: true},
	{Input: bareInterest, FragCount: 1, PayloadL: payloadInterestL},
	{Input: "6409 payload=" + payloadInterest, FragCount: 1, PayloadL: payloadInterestL},
	{Input: "6402 unknown-critical=6300", Bad: true},
	{Input: "6404 unknown-critical=FD03BF00", Bad: true},
	{Input: "6404 unknown-ignored=FD03BC00", FragCount: 1},
	{Input: "6413 seq=5108A0A1A2A3A4A5A600 fragcount=530102 payload=" + payloadFragment,
		SeqNum: 0xA0A1A2A3A4A5A600, FragIndex: 0, FragCount: 2,
		PayloadL: payloadFragmentL},
	{Input: "6416 seq=5108A0A1A2A3A4A5A601 fragindex=520101 fragcount=530102 " +
		"payload=" + payloadFragment,
		SeqNum: 0xA0A1A2A3A4A5A601, FragIndex: 1, FragCount: 2,
		PayloadL: payloadFragmentL},
	{Input: "6417 seq=5108A0A1A2A3A4A5A601 fragindex=520102 fragcount=530102 " +
		"payload=" + payloadFragment, Bad: true}, // FragIndex >= FragCount
	{Input: "6413 pittoken=62089A414B412BC38EB2 payload=" + payloadInterest,
		FragCount: 1, PitToken: 0x9A414B412BC38EB2, PayloadL: payloadInterestL},
	{Input: "6406 pittoken=620420A3C0D7", Bad: true}, // PitToken is not 8-octet
	{Input: "640D nack=FD032000(noreason) payload=" + payloadInterest,
		FragCount: 1, NackReason: an.NackUnspecified, PayloadL: payloadInterestL},
	{Input: "6412 nack=FD032005(FD03210196~noroute) payload=" + payloadInterest,
		FragCount: 1, NackReason: an.NackNoRoute, PayloadL: payloadInterestL},
	{Input: "640E congmark=FD03400104 payload=" + payloadInterest,
		FragCount: 1, CongMark: 4, PayloadL: payloadInterestL},
}
