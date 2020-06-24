package an

import "strconv"

// NackReason assigned numbers.
const (
	NackNone        = 0
	NackCongestion  = 50
	NackDuplicate   = 100
	NackNoRoute     = 150
	NackUnspecified = 255

	_ = "enumgen:NackReason"
)

// NackReasonString converts NackReason to string.
func NackReasonString(reason uint8) string {
	switch reason {
	case NackNone:
		return "none"
	case NackCongestion:
		return "congestion"
	case NackDuplicate:
		return "duplicate"
	case NackNoRoute:
		return "no-route"
	case NackUnspecified:
		return "unspecified"
	}
	return strconv.Itoa(int(reason))
}
