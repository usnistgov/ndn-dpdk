package socketface

import (
	"net"

	"ndn-dpdk/iface"
)

type SocketFace interface {
	iface.Face
}

func New(conn net.Conn) SocketFace {
	if pktConn, ok := conn.(net.PacketConn); ok {
		return NewDatagramFace(pktConn)
	}
	return NewStreamFace(conn)
}

type baseFace struct {
	id   iface.FaceId
	conn net.Conn
}

func (f *baseFace) GetFaceId() iface.FaceId {
	return f.id
}

func (f *baseFace) Close() error {
	return f.conn.Close()
}
