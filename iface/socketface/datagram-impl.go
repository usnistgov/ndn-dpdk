package socketface

import "C"
import (
	"errors"
	"fmt"
	"net"

	"ndn-dpdk/dpdk"
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
		seg0 := pkt.GetFirstSegment()
		seg0.SetHeadroom(0)

		buf := seg0.AsByteSlice()
		buf = buf[:cap(buf)]
		nOctets, e := face.GetConn().Read(buf)
		if e != nil {
			if face.handleError("RX", e) {
				pkt.Close()
				return
			}
			continue
		}
		seg0.Append(buf[:nOctets])

		face.rxPkt(pkt)
	}
}

func (datagramImpl) Send(face *SocketFace, pkt dpdk.Packet) error {
	var buf []byte
	if pkt.CountSegments() > 1 {
		buf = pkt.ReadAll()
	} else {
		buf = pkt.GetFirstSegment().AsByteSlice()
	}
	_, e := face.GetConn().Write(buf)
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
