package socketface

import "C"
import (
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/faceuri"
)

// SocketFace implementation for datagram-oriented sockets.
type datagramImpl struct{}

func (datagramImpl) RxLoop(face *SocketFace) {
	for {
		mbuf, e := face.rxMp.Alloc()
		if e != nil {
			face.logger.WithError(e).Error("RX alloc error")
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

func (datagramImpl) Send(face *SocketFace, pkt dpdk.Packet) error {
	var buf []byte
	if pkt.CountSegments() > 1 {
		buf = pkt.ReadAll()
	} else {
		buf = pkt.GetFirstSegment().AsByteSlice()
	}
	_, e := face.conn.Write(buf)
	return e
}

type udpImpl struct {
	datagramImpl
}

func (udpImpl) FormatFaceUri(addr net.Addr) *faceuri.FaceUri {
	a := addr.(*net.UDPAddr)
	if a.IP.To4() == nil {
		// FaceUri cannot represent IPv6 address
		return faceuri.MustParse(fmt.Sprintf("udp4://192.0.2.6:%d", a.Port))
	}
	return faceuri.MustParse(fmt.Sprintf("udp4://%s", a))
}

type unixgramImpl struct {
	datagramImpl
}

func (unixgramImpl) FormatFaceUri(addr net.Addr) *faceuri.FaceUri {
	// FaceUri cannot represent non-UDP
	return faceuri.MustParse("udp4://192.0.2.0:1")
}

func init() {
	implByNetwork["udp"] = udpImpl{}
	implByNetwork["udp4"] = udpImpl{}
	implByNetwork["udp6"] = udpImpl{}
	implByNetwork["unixgram"] = unixgramImpl{}
}
