package ndni

//go:generate stringer -type=NdnError,L2PktType,L3PktType,DataSatisfyResult -output=enum_string.go
//go:generate go run ../mk/enumgen/ -guard=NDN_DPDK_NDN_ENUM_H -out=../csrc/ndn/enum.h .

const (
	// NameMaxLength is the maximum TLV-LENGTH for Name.
	NameMaxLength = 2048

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

	// InterestEstimatedHeadroom is a safe headroom to encode Interest.
	InterestEstimatedHeadroom = 1 + 5 // Interest TL

	// InterestEstimatedTailroom is a safe tailroom to encode Interest.
	InterestEstimatedTailroom = 4 + NameMaxLength + InterestTemplateBufLen

	// DataEstimatedHeadroom is a safe headroom to encode Data.
	DataEstimatedHeadroom = 1 + 5 // Data TL

	// DataEstimatedTailroom is a safe tailroom to encode Data, excluding payload.
	DataEstimatedTailroom = 0 +
		1 + 3 + NameMaxLength + // Name
		1 + 1 + 1 + 1 + 4 + // MetaInfo with FreshnessPeriod
		1 + 3 + 0 + // Content TL
		39 // Signature

	_ = "enumgen"
)

// NdnError indicates an error condition.
type NdnError int

// NdnError values.
const (
	NdnErrOK NdnError = iota
	NdnErrIncomplete
	NdnErrAllocError
	NdnErrLengthOverflow
	NdnErrBadType
	NdnErrBadNni
	NdnErrFragmented
	NdnErrUnknownCriticalLpHeader
	NdnErrFragIndexExceedFragCount
	NdnErrLpHasTrailer
	NdnErrBadLpSeqNum
	NdnErrBadPitToken
	NdnErrNameIsEmpty
	NdnErrNameTooLong
	NdnErrBadNameComponentType
	NdnErrNameHasComponentAfterDigest
	NdnErrBadDigestComponentLength
	NdnErrBadNonceLength
	NdnErrBadInterestLifetime
	NdnErrBadHopLimitLength
	NdnErrHopLimitZero

	_ = "enumgen:NdnError"
)

func (e NdnError) Error() string {
	return e.String()
}

// L2PktType indicates layer 2 packet type.
type L2PktType int

// L2PktType values.
const (
	L2PktTypeNone L2PktType = iota
	L2PktTypeNdnlpV2

	_ = "enumgen:L2PktType"
)

// L3PktType indicates layer 3 packet type.
type L3PktType int

// L3PktType values.
const (
	L3PktTypeNone L3PktType = iota
	L3PktTypeInterest
	L3PktTypeData
	L3PktTypeNack
	L3PktTypeMAX

	_ = "enumgen:L3PktType"
)

// DataSatisfyResult indicates the result of data.CanSatisfy function.
type DataSatisfyResult int

// DataSatisfyResult values.
const (
	DataSatisfyYes        DataSatisfyResult = 0 // Data satisfies Interest
	DataSatisfyNo         DataSatisfyResult = 1 // Data does not satisfy Interest
	DataSatisfyNeedDigest DataSatisfyResult = 2 // need Data digest to determine

	_ = "enumgen:DataSatisfyResult"
)
