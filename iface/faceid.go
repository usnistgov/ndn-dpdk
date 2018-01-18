package iface

type FaceKind int

const (
	FaceKind_None FaceKind = iota
	FaceKind_EthDev
	FaceKind_Udp
	FaceKind_Socket
)

// Numeric face identifier, may appear in rte_mbuf.port field
type FaceId uint16

const (
	FACEID_INVALID FaceId = 0
	FACEID_MIN     FaceId = 1
	FACEID_MAX     FaceId = 0xFFFF
)

func (id FaceId) GetKind() FaceKind {
	switch id >> 12 {
	case 0x1:
		return FaceKind_EthDev
	case 0x4:
	case 0x5:
		return FaceKind_Udp
	case 0xE:
		return FaceKind_Socket
	}
	return FaceKind_None
}
