package socketface

import (
	"net"

	"ndn-dpdk/ndn"
)

type DatagramFace struct {
	baseFace
}

func NewDatagramFace(conn net.PacketConn) (face *DatagramFace) {
	panic("not implemented")
}

func (face *DatagramFace) RxBurst(pkts []ndn.Packet) int {
	panic("not implemented")
}

func (face *DatagramFace) TxBurst(pkts []ndn.Packet) {
	panic("not implemented")
}
