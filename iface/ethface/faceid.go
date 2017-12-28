package ethface

import (
	"ndn-dpdk/iface"
)

func hasValidFaceId(port uint16) bool {
	return port < 0x1000
}

func FaceIdFromEthDev(port uint16) iface.FaceId {
	if !hasValidFaceId(port) {
		return iface.FACEID_INVALID
	}
	return iface.FaceId(0x1000 | port)
}
