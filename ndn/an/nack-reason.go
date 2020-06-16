package an

// NackReason indicates a Nack reason.
type NackReason uint8

// Known Nack reasons.
const (
	NackNone        NackReason = 0
	NackCongestion  NackReason = 50
	NackDuplicate   NackReason = 100
	NackNoRoute     NackReason = 150
	NackUnspecified NackReason = 255
)
