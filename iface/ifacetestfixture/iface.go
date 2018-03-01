package ifacetestfixture

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type IBaseFace interface {
	GetFaceId() iface.FaceId
	ReadCounters() iface.Counters
}

type IRxFace interface {
	IBaseFace
}

type ITxFace interface {
	IBaseFace
	TxBurst(pkts []ndn.Packet)
}
