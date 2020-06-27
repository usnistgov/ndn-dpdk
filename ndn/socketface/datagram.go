package socketface

import "C"
import (
	"fmt"
	"net"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
)

type udpImpl struct {
	nopRedialer
}

func (udpImpl) Dial(network, local, remote string) (net.Conn, error) {
	raddr, e := net.ResolveUDPAddr(network, remote)
	if e != nil {
		return nil, fmt.Errorf("resolve remote %w", e)
	}
	laddr := &net.UDPAddr{Port: raddr.Port}
	if local != "" {
		if laddr, e = net.ResolveUDPAddr(network, local); e != nil {
			return nil, fmt.Errorf("resolve local %w", e)
		}
	}
	return net.DialUDP(network, laddr, raddr)
}

func (udpImpl) RxLoop(face *SocketFace) {
	for {
		buffer := make([]byte, face.cfg.RxBufferLength)
		datagramLength, e := face.GetConn().Read(buffer)
		if e != nil {
			if face.handleError(e) {
				return
			}
			continue
		}
		wire := buffer[:datagramLength]

		var packet ndn.Packet
		e = tlv.Decode(wire, &packet)
		if e != nil { // ignore decoding error
			continue
		}
		face.rx <- &packet
	}
}

func init() {
	var udp udpImpl
	implByNetwork["udp"] = udp
	implByNetwork["udp4"] = udp
	implByNetwork["udp6"] = udp
}
