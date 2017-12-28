package iface

import (
	"ndn-dpdk/ndn"
)

type baseFace interface {
	GetFaceId() FaceId
	Close() error
}

type rxFaceMethods interface {
	RxBurst(pkts []ndn.Packet) int
}

type RxFace interface {
	baseFace
	rxFaceMethods
}

type txFaceMethods interface {
	TxBurst(pkts []ndn.Packet)
}

type TxFace interface {
	baseFace
	txFaceMethods
}

type Face interface {
	baseFace
	rxFaceMethods
	txFaceMethods
}
