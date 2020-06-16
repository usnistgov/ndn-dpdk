package an

// TlvType indicates a TLV-TYPE number.
type TlvType uint32

// Known TLV-TYPE numbers.
const (
	TtInvalid TlvType = 0x00

	TtLpPacket       TlvType = 0x64
	TtLpPayload      TlvType = 0x50
	TtLpSeqNo        TlvType = 0x51
	TtFragIndex      TlvType = 0x52
	TtFragCount      TlvType = 0x53
	TtPitToken       TlvType = 0x62
	TtNack           TlvType = 0x0320
	TtNackReason     TlvType = 0x0321
	TtCongestionMark TlvType = 0x0340

	TtName                            TlvType = 0x07
	TtGenericNameComponent            TlvType = 0x08
	TtImplicitSha256DigestComponent   TlvType = 0x01
	TtParametersSha256DigestComponent TlvType = 0x02
	TtKeywordNameComponent            TlvType = 0x20
	TtSegmentNameComponent            TlvType = 0x21
	TtByteOffsetNameComponent         TlvType = 0x22
	TtVersionNameComponent            TlvType = 0x23
	TtTimestampNameComponent          TlvType = 0x24
	TtSequenceNumNameComponent        TlvType = 0x25

	TtInterest               TlvType = 0x05
	TtCanBePrefix            TlvType = 0x21
	TtMustBeFresh            TlvType = 0x12
	TtForwardingHint         TlvType = 0x1E
	TtDelegation             TlvType = 0x1F
	TtPreference             TlvType = 0x1E
	TtNonce                  TlvType = 0x0A
	TtInterestLifetime       TlvType = 0x0C
	TtHopLimit               TlvType = 0x22
	TtApplicationParameters  TlvType = 0x24
	TtInterestSignatureInfo  TlvType = 0x2C
	TtInterestSignatureValue TlvType = 0x2E

	TtData            TlvType = 0x06
	TtMetaInfo        TlvType = 0x14
	TtContentType     TlvType = 0x18
	TtFreshnessPeriod TlvType = 0x19
	TtFinalBlock      TlvType = 0x1A
	TtContent         TlvType = 0x15
	TtSignatureInfo   TlvType = 0x16
	TtSignatureValue  TlvType = 0x17

	TtSignatureType   TlvType = 0x1B
	TtKeyLocator      TlvType = 0x1C
	TtKeyDigest       TlvType = 0x1D
	TtSignatureNonce  TlvType = 0x26
	TtSignatureTime   TlvType = 0x28
	TtSignatureSeqNum TlvType = 0x2A
)
