package socketface

import "C"
import (
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/ndn"
)

// SocketFace implementation for stream-oriented sockets.
type streamImpl struct{}

func (impl streamImpl) RxLoop(face *SocketFace) {
	buf := make(ndn.TlvBytes, face.rxMp.GetDataroom())
	nAvail := 0
	for {
		nRead, e := face.GetConn().Read(buf[nAvail:])
		if e != nil {
			if face.handleError("RX", e) {
				return
			}
			continue
		}
		nAvail += nRead

		// parse and post packets
		offset := 0
		for {
			n := impl.postPacket(face, buf[offset:nAvail])
			if n == 0 {
				break
			}
			offset += n
		}

		// move remaining portion to the front
		for i := offset; i < nAvail; i++ {
			buf[i-offset] = buf[i]
		}
		nAvail -= offset
	}
}

func (streamImpl) postPacket(face *SocketFace, buf ndn.TlvBytes) (n int) {
	element, _ := buf.ExtractElement()
	if element == nil {
		return 0
	}

	mbuf, e := face.rxMp.Alloc()
	if e != nil {
		face.logger.WithError(e).Error("RX alloc error")
		return n
	}

	pkt := mbuf.AsPacket()
	pkt.SetTimestamp(dpdk.TscNow())
	seg0 := pkt.GetFirstSegment()
	seg0.SetHeadroom(0)
	seg0.Append([]byte(element))

	face.rxPkt(pkt)
	return len(element)
}

func (streamImpl) Send(face *SocketFace, pkt dpdk.Packet) error {
	for seg, ok := pkt.GetFirstSegment(), true; ok; seg, ok = seg.GetNext() {
		buf := seg.AsByteSlice()
		_, e := face.GetConn().Write(buf)
		if e != nil {
			return e
		}
	}
	return nil
}

type tcpImpl struct {
	streamImpl
	localAddrRedialer
}

func (tcpImpl) FormatFaceUri(addr net.Addr) *faceuri.FaceUri {
	a := addr.(*net.TCPAddr)
	if a.IP.To4() == nil {
		// FaceUri cannot represent IPv6 address
		return faceuri.MustParse(fmt.Sprintf("tcp4://192.0.2.6:%d", a.Port))
	}
	return faceuri.MustParse(fmt.Sprintf("tcp4://%s", a))
}

type unixImpl struct {
	streamImpl
	noLocalAddrRedialer
}

func (unixImpl) FormatFaceUri(addr net.Addr) *faceuri.FaceUri {
	a := addr.(*net.UnixAddr)
	if a.Name == "@" {
		return faceuri.MustParse("unix:///invalid")
	}
	return faceuri.MustParse(fmt.Sprintf("unix://%s", a.Name))
}

func init() {
	implByNetwork["tcp"] = tcpImpl{}
	implByNetwork["tcp4"] = tcpImpl{}
	implByNetwork["tcp6"] = tcpImpl{}
	implByNetwork["unix"] = unixImpl{}
}
