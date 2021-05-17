package ndntestvector

import (
	"github.com/usnistgov/ndn-dpdk/core/testenv"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
)

const (
	bareInterest     = "0505 0703080141"
	payloadInterest  = "5007 " + bareInterest
	payloadInterestL = 7
	payloadFragment  = "5004 D0D1D2D3"
	payloadFragmentL = 4
)

var bytesFromHex = testenv.BytesFromHex

// LpDecodeTests contains test vectors for NDNLPv2 decoder.
var LpDecodeTests = []struct {
	Input      string
	Bad        bool
	SeqNum     uint64
	FragIndex  uint16
	FragCount  uint16
	PitToken   []byte
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
		PitToken: bytesFromHex("9A414B412BC38EB2"), PayloadL: payloadInterestL},
	{Input: "640F pittoken=620420A3C0D7 payload=" + payloadInterest,
		PitToken: bytesFromHex("20A3C0D7"), PayloadL: payloadInterestL},
	{Input: "642B pittoken=6220B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7 " +
		"payload=" + payloadInterest,
		PitToken: bytesFromHex("B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7"),
		PayloadL: payloadInterestL},
	{Input: "642C pittoken=6221B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7B0B1B2B3B4B5B6B7BB " +
		"payload=" + payloadInterest, Bad: true}, // PitToken too long
	{Input: "640D nack=FD032000(noreason) payload=" + payloadInterest,
		NackReason: an.NackUnspecified, PayloadL: payloadInterestL},
	{Input: "6412 nack=FD032005(FD03210196~noroute) payload=" + payloadInterest,
		NackReason: an.NackNoRoute, PayloadL: payloadInterestL},
	{Input: "640E congmark=FD03400104 payload=" + payloadInterest,
		CongMark: 4, PayloadL: payloadInterestL},
}
