package ndni

import (
	"crypto/sha256"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn"
)

//go:generate go run ../mk/enumgen/ -guard=NDN_DPDK_NDNI_ENUM_H -out=../csrc/ndni/enum.h .
//go:generate go run ../mk/enumgen/ -guard=NDN_DPDK_NDNI_AN_H -out=../csrc/ndni/an.h ../ndn/an

const (
	// NameMaxLength is the maximum TLV-LENGTH for Name.
	NameMaxLength = 2048

	// ImplicitDigestLength is the TLV-LENGTH of ImplicitDigestNameComponent.
	ImplicitDigestLength = 32

	// ImplicitDigestSize is size of of ImplicitDigestNameComponent TLV.
	ImplicitDigestSize = 34

	// PNameCachedComponents is the number of cached component boundaries and hashes in PName struct.
	PNameCachedComponents = 17

	// PInterestMaxFwHints is the maximum number of decoded forwarding hints on Interest.
	// Additional forwarding hints are ignored.
	PInterestMaxFwHints = 4

	// DefaultInterestLifetime is the default value of InterestLifetime.
	DefaultInterestLifetime = 4000

	// LpHeaderEstimatedHeadroom is a safe headroom to prepend NDNLPv2 header.
	LpHeaderEstimatedHeadroom = 0 +
		1 + 5 + // LpPacket TL
		1 + 1 + 8 + // SeqNo
		1 + 1 + 2 + // FragIndex
		1 + 1 + 2 + // FragCount
		1 + 1 + 8 + // PitToken
		3 + 1 + 3 + 1 + 1 + // Nack
		3 + 1 + 1 + // CongestionMark
		1 + 5 // Payload TL

	// InterestTemplateBufLen is the buffer length for InterestTemplate.
	// It can accommodate two forwarding hints.
	InterestTemplateBufLen = 2*NameMaxLength + 256

	// InterestTemplateDataroom is a safe buffer size to encode Interest.
	InterestTemplateDataroom = 0 +
		1 + 5 + // Interest TL
		1 + 3 + NameMaxLength + // Name
		InterestTemplateBufLen // other fields

	// DataGenBufLen is the buffer length for DataGen.
	DataGenBufLen = 0 +
		1 + 3 + NameMaxLength + // Name suffix
		1 + 1 + 1 + 1 + 4 + // MetaInfo with FreshnessPeriod
		1 + 3 + 0 + // Content TL
		39 // Signature

	// DataGenDataroom is a safe buffer size to encode Data.
	DataGenDataroom = 0 +
		1 + 5 + // Data TL
		1 + 3 + NameMaxLength // Name prefix

	_ = "enumgen"
)

func _() {
	var x [1]int
	x[ndn.DefaultInterestLifetime-(DefaultInterestLifetime*time.Millisecond)] = 0
	x[sha256.Size-ImplicitDigestLength] = 0
}

// PktType indicates packet type in mbuf.
type PktType int

// PktType values.
const (
	PktFragment PktType = iota
	PktInterest         // Interest
	PktData             // Data
	PktNack             // Nack
	_
	PktSInterest // Interest unparsed
	PktSData     // Data unparsed
	PktSNack     // Nack unparsed

	PktMax = PktNack + 1 // maximum excluding slim types

	_ = "enumgen:PktType"
)

// DataSatisfyResult indicates the result of Data.CanSatisfy function.
type DataSatisfyResult int

// DataSatisfyResult values.
const (
	DataSatisfyYes        DataSatisfyResult = 0 // Data satisfies Interest
	DataSatisfyNo         DataSatisfyResult = 1 // Data does not satisfy Interest
	DataSatisfyNeedDigest DataSatisfyResult = 2 // need Data digest to determine

	_ = "enumgen:DataSatisfyResult"
)
