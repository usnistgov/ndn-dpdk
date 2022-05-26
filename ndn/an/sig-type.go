package an

import "strconv"

// SigType assigned numbers.
const (
	SigSha256          = 0x00
	SigSha256WithRsa   = 0x01
	SigSha256WithEcdsa = 0x03
	SigHmacWithSha256  = 0x04
	SigEd25519         = 0x05
	SigNull            = 0xC8

	_ = "enumgen:SigType"
)

// SigTypeString converts SigType to string.
func SigTypeString(sigType uint32) string {
	switch sigType {
	case SigSha256:
		return "SHA256"
	case SigSha256WithRsa:
		return "RSA"
	case SigSha256WithEcdsa:
		return "ECDSA"
	case SigHmacWithSha256:
		return "HMAC"
	case SigEd25519:
		return "Ed25519"
	case SigNull:
		return "null"
	}
	return strconv.Itoa(int(sigType))
}
