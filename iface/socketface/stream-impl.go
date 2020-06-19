package socketface

import "C"
import (
	"net"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

// SocketFace implementation for stream-oriented sockets.
type streamImpl struct{}

func (impl streamImpl) RxLoop(face *SocketFace) {
	buf := make([]byte, face.rxMp.GetDataroom())
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
		d := tlv.Decoder(buf[:nAvail])
		elements := d.Elements()
		if len(elements) == 0 {
			continue
		}

		vec, e := face.rxMp.Alloc(len(elements))
		if e == nil {
			for i, de := range elements {
				pkt := vec[i]
				pkt.SetHeadroom(0)
				pkt.Append(de.Wire)
				face.rxPkt(pkt)
			}
		} else {
			face.logger.WithError(e).Error("RX alloc error")
		}

		// move remaining portion to the front
		nAvail = copy(buf, d.Rest())
	}
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
