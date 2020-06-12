package socketface

import "C"
import (
	"net"

	"ndn-dpdk/dpdk/pktmbuf"
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

	vec, e := face.rxMp.Alloc(1)
	if e != nil {
		face.logger.WithError(e).Error("RX alloc error")
		return n
	}

	pkt := vec[0]
	pkt.SetHeadroom(0)
	pkt.Append([]byte(element))
	face.rxPkt(pkt)
	return len(element)
}

func (streamImpl) Send(face *SocketFace, pkt *pktmbuf.Packet) error {
	_, e := face.GetConn().Write(pkt.ReadAll())
	if e != nil {
		return e
	}
	return nil
}

type tcpImpl struct {
	streamImpl
	noLocalAddrDialer
	localAddrRedialer
}

func (tcpImpl) ValidateAddr(network, address string, isLocal bool) error {
	_, e := net.ResolveTCPAddr(network, address)
	return e
}

type unixImpl struct {
	streamImpl
	unixAddrValidator
	noLocalAddrDialer
	noLocalAddrRedialer
}
