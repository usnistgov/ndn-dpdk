package an

// SigType assigned numbers.
const (
	SigSha256          = 0x00
	SigSha256WithRsa   = 0x01
	SigSha256WithEcdsa = 0x03
	SigHmacWithSha256  = 0x04
	SigNull            = 0xC8

	_ = "enumgen:SigType"
)
