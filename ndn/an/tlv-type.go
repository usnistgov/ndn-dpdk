package an

// TLV-TYPE assigned numbers.
const (
	TtInvalid = 0x00

	TtLpPacket       = 0x64
	TtLpPayload      = 0x50
	TtLpSeqNum       = 0x51
	TtFragIndex      = 0x52
	TtFragCount      = 0x53
	TtPitToken       = 0x62
	TtNack           = 0x0320
	TtNackReason     = 0x0321
	TtCongestionMark = 0x0340

	TtName                            = 0x07
	TtGenericNameComponent            = 0x08
	TtImplicitSha256DigestComponent   = 0x01
	TtParametersSha256DigestComponent = 0x02
	TtKeywordNameComponent            = 0x20
	TtSegmentNameComponent            = 0x32
	TtByteOffsetNameComponent         = 0x34
	TtVersionNameComponent            = 0x36
	TtTimestampNameComponent          = 0x38
	TtSequenceNumNameComponent        = 0x3A

	TtInterest         = 0x05
	TtCanBePrefix      = 0x21
	TtMustBeFresh      = 0x12
	TtForwardingHint   = 0x1E
	TtNonce            = 0x0A
	TtInterestLifetime = 0x0C
	TtHopLimit         = 0x22
	TtAppParameters    = 0x24
	TtISigInfo         = 0x2C
	TtISigValue        = 0x2E

	TtData            = 0x06
	TtMetaInfo        = 0x14
	TtContentType     = 0x18
	TtFreshnessPeriod = 0x19
	TtFinalBlock      = 0x1A
	TtContent         = 0x15
	TtDSigInfo        = 0x16
	TtDSigValue       = 0x17

	TtSigType    = 0x1B
	TtKeyLocator = 0x1C
	TtKeyDigest  = 0x1D
	TtSigNonce   = 0x26
	TtSigTime    = 0x28
	TtSigSeqNum  = 0x2A

	TtValidityPeriod = 0x00FD
	TtNotBefore      = 0x00FE
	TtNotAfter       = 0x00FF

	_ = "enumgen"
)
