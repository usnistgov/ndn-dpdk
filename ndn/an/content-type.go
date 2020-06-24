package an

// ContentType assigned numbers.
const (
	ContentBlob      = 0x00
	ContentLink      = 0x01
	ContentKey       = 0x02
	ContentNack      = 0x03
	ContentPrefixAnn = 0x05

	_ = "enumgen:ContentType"
)
