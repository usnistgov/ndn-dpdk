package sockettransport

import (
	"fmt"
	"net"
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

func (udpImpl) RxLoop(tr *Transport) {
	for {
		buffer := make([]byte, tr.cfg.RxBufferLength)
		datagramLength, e := tr.GetConn().Read(buffer)
		if e != nil {
			if tr.handleError(e) {
				return
			}
			continue
		}

		wire := buffer[:datagramLength]
		tr.rx <- wire
	}
}

func init() {
	var udp udpImpl
	implByNetwork["udp"] = udp
	implByNetwork["udp4"] = udp
	implByNetwork["udp6"] = udp
}
