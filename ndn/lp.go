package ndn

import (
	"encoding/binary"

	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// LpHeader contains information in NDNLPv2 header.
type LpHeader struct {
	PitToken   []byte
	NackReason uint8
	CongMark   int
}

// Empty returns true if LpHeader has zero fields.
func (lph LpHeader) Empty() bool {
	return len(lph.PitToken) == 0 && lph.NackReason == an.NackNone && lph.CongMark == 0
}

func (lph LpHeader) encode() (fields []interface{}) {
	if len(lph.PitToken) > 0 {
		fields = append(fields, tlv.MakeElement(an.TtPitToken, lph.PitToken))
	}
	if lph.NackReason != an.NackNone {
		var nackV []byte
		if lph.NackReason != an.NackUnspecified {
			nackV, _ = tlv.Encode(tlv.MakeElementNNI(an.TtNackReason, lph.NackReason))
		}
		fields = append(fields, tlv.MakeElement(an.TtNack, nackV))
	}
	if lph.CongMark != 0 {
		fields = append(fields, tlv.MakeElementNNI(an.TtCongestionMark, lph.CongMark))
	}
	return fields
}

func (lph *LpHeader) inheritFrom(src LpHeader) {
	lph.PitToken = src.PitToken
	lph.CongMark = src.CongMark
}

// PitTokenFromUint creates a PIT token from uint64, interpreted as little endian.
func PitTokenFromUint(n uint64) []byte {
	token := make([]byte, 8)
	binary.LittleEndian.PutUint64(token, n)
	return token
}

// PitTokenToUint reads a 8-octet PIT token as uint64, interpreted as little endian.
// Returns 0 if the input token is not 8 octets.
func PitTokenToUint(token []byte) uint64 {
	if len(token) != 8 {
		return 0
	}
	return binary.LittleEndian.Uint64(token)
}

func lpIsCritical(typ uint32) bool {
	return typ < 800 || typ > 959 && (typ&0x03) != 0
}
