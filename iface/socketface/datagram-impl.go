package socketface

import "C"
import (
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/faceuri"
)

// SocketFace implementation for datagram-oriented sockets.
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
	for {
		mbuf, e := face.rxMp.Alloc()
		if e != nil {
			face.logger.Printf("RX alloc error: %v", e)
			continue
		}

		pkt := mbuf.AsPacket()
		pkt.SetTimestamp(dpdk.TscNow())
		seg0 := pkt.GetFirstSegment()
		seg0.SetHeadroom(0)

		buf := seg0.AsByteSlice()
		buf = buf[:cap(buf)]
		nOctets, e := face.conn.Read(buf)
		if e != nil {
			if face.handleError("RX", e) {
				pkt.Close()
				return
			}
			continue
		}
		seg0.Append(buf[:nOctets])

		select {
		case <-face.rxQuit:
			pkt.Close()
			return
		case face.rxQueue <- pkt:
		default:
			pkt.Close()
			face.rxReportCongestion()
		}
	}
}

func (impl *datagramImpl) Send(pkt dpdk.Packet) error {
	var buf []byte
	if pkt.CountSegments() > 1 {
		buf = pkt.ReadAll()
	} else {
		buf = pkt.GetFirstSegment().AsByteSlice()
	}
	_, e := impl.face.conn.Write(buf)
	return e
}

func (impl *datagramImpl) FormatFaceUri(addr net.Addr) *faceuri.FaceUri {
	if a, ok := addr.(*net.UDPAddr); ok {
		if a.IP.To4() != nil {
			return faceuri.MustParse(fmt.Sprintf("udp4://%s", a))
		} else {
			// FaceUri cannot represent IPv6 address
			return faceuri.MustParse(fmt.Sprintf("udp4://192.0.2.6:%d", a.Port))
		}
	}
	// FaceUri cannot represent non-UDP
	return faceuri.MustParse("udp4://192.0.2.0:1")
}
