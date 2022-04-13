package sockettransport

import (
	"fmt"
	"net"

	"github.com/gogf/greuse"
)

type datagramImpl struct {
	nopRedialer
}

func (datagramImpl) Read(tr *transport, trc *trConn, buf []byte) (n int, e error) {
	return trc.conn.Read(buf)
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
