package socketface

import "C"
import (
	"net"

	"ndn-dpdk/dpdk"
)

type datagramImpl struct {
	face *SocketFace
}

func newDatagramImpl(face *SocketFace, conn net.PacketConn) *datagramImpl {
	impl := new(datagramImpl)
	impl.face = face
	return impl
}

func (impl *datagramImpl) Recv() ([]byte, error) {
	buf := make([]byte, impl.face.rxMp.GetDataroom())
	nOctets, e := impl.face.conn.Read(buf)
	if e != nil {
		return nil, e
	}
	return buf[:nOctets], nil
}

func (impl *datagramImpl) Send(pkt dpdk.Packet) error {
	buf := make([]byte, pkt.Len())
	pkt.ReadTo(0, buf)
	_, e := impl.face.conn.Write(buf)
	return e
}
