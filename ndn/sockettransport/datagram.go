package sockettransport

import (
	"fmt"
	"net"
)

type datagramImpl struct {
	nopRedialer
}

func (datagramImpl) RxLoop(tr *transport) error {
	for {
		buffer := make([]byte, tr.cfg.RxBufferLength)
		datagramLength, e := tr.Conn().Read(buffer)
		if e != nil {
			return e
		}

		wire := buffer[:datagramLength]
		tr.rx <- wire
	}
}

type pipeImpl struct {
	datagramImpl
}

func (pipeImpl) Dial(network, local, remote string) (net.Conn, error) {
	return nil, fmt.Errorf("cannot dial %s", network)
}

type udpImpl struct {
	datagramImpl
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

func init() {
	implByNetwork["pipe"] = pipeImpl{}

	implByNetwork["udp"] = udpImpl{}
	implByNetwork["udp4"] = udpImpl{}
	implByNetwork["udp6"] = udpImpl{}
}
