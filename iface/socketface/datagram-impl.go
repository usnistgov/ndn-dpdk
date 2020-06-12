package socketface

import "C"
import (
	"errors"
	"fmt"
	"net"

	"ndn-dpdk/dpdk/pktmbuf"
)

// SocketFace implementation for datagram-oriented sockets.
type datagramImpl struct{}

func (datagramImpl) RxLoop(face *SocketFace) {
	buf := make([]byte, face.rxMp.GetDataroom())
	for {
		nOctets, e := face.GetConn().Read(buf)
		if e != nil {
			if face.handleError("RX", e) {
				return
			}
			continue
		}

		vec, e := face.rxMp.Alloc(1)
		if e != nil {
			face.logger.WithError(e).Error("RX alloc error")
			continue
		}

		pkt := vec[0]
		pkt.SetHeadroom(0)
		pkt.Append(buf[:nOctets])
		face.rxPkt(pkt)
	}
}

func (datagramImpl) Send(face *SocketFace, pkt *pktmbuf.Packet) error {
	_, e := face.GetConn().Write(pkt.ReadAll())
	return e
}

type udpImpl struct {
	datagramImpl
	nopRedialer
}

func (udpImpl) ValidateAddr(network, address string, isLocal bool) error {
	_, e := net.ResolveUDPAddr(network, address)
	return e
}

func (udpImpl) Dial(network, local, remote string) (net.Conn, error) {
	raddr, e := net.ResolveUDPAddr(network, remote)
	if e != nil {
		return nil, fmt.Errorf("Remote: %v", e)
	}
	laddr := &net.UDPAddr{Port: raddr.Port}
	if local != "" {
		if laddr, e = net.ResolveUDPAddr(network, local); e != nil {
			return nil, fmt.Errorf("Local: %v", e)
		}
	}
	return net.DialUDP(network, laddr, raddr)
}

type unixgramImpl struct {
	datagramImpl
	unixAddrValidator
	noLocalAddrDialer
	noLocalAddrRedialer
}

func (unixgramImpl) Dial(network, local, remote string) (net.Conn, error) {
	return nil, errors.New("not implemented")
}
