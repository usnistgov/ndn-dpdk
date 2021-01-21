package sockettransport

import (
	"fmt"
	"net"

	"github.com/gogf/greuse"
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
		tr.p.Rx <- wire
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
	return greuse.Dial(network, local, remote)
}

func init() {
	implByNetwork["pipe"] = pipeImpl{}

	implByNetwork["udp"] = udpImpl{}
	implByNetwork["udp4"] = udpImpl{}
	implByNetwork["udp6"] = udpImpl{}
}
