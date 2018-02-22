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

func (impl *datagramImpl) RxLoop() {
	face := impl.face
	buf := make([]byte, face.rxMp.GetDataroom())
	for {
		nOctets, e := face.conn.Read(buf)
		if face.handleError("RX", e) {
			return
		}

		mbuf, e := face.rxMp.Alloc()
		if e != nil {
			face.logger.Printf("RX alloc error: %v", e)
			continue
		}

		pkt := mbuf.AsPacket()
		pkt.SetPort(uint16(impl.face.GetFaceId()))
		pkt.SetTimestamp(dpdk.TscNow())
		seg0 := pkt.GetFirstSegment()
		seg0.SetHeadroom(0)
		seg0.Append(buf[:nOctets])

		select {
		case <-face.rxQuit:
			pkt.Close()
			return
		case face.rxQueue <- pkt:
		default:
			pkt.Close()
			face.rxCongestions++
			face.logger.Printf("RX queue is full, %d", face.rxCongestions)
		}
	}
}

func (impl *datagramImpl) Send(pkt dpdk.Packet) error {
	buf := make([]byte, pkt.Len())
	pkt.ReadTo(0, buf)
	_, e := impl.face.conn.Write(buf)
	return e
}
