package sockettransport

import (
	"net"

	"github.com/gogf/greuse"
)

type impl interface {
	// Dial the socket.
	Dial(network, local, remote string) (net.Conn, error)

	// Redial the socket.
	Redial(oldConn net.Conn) (net.Conn, error)

	// Receive a TLV packet on the socket.
	Read(tr *transport, buf []byte) (n int, e error)
}

var implByNetwork = map[string]impl{}

// noLocalAddrDialer dials with only remote addr.
type noLocalAddrDialer struct{}

func (noLocalAddrDialer) Dial(network, _, remote string) (net.Conn, error) {
	dialer := net.Dialer{
		Control: greuse.Control,
	}
	return dialer.Dial(network, remote)
}

// localAddrRedialer redials reusing local addr.
type localAddrRedialer struct{}

func (localAddrRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	local, remote := oldConn.LocalAddr(), oldConn.RemoteAddr()
	oldConn.Close() // ignore error

	dialer := net.Dialer{
		LocalAddr: local,
		Control:   greuse.Control,
	}
	return dialer.Dial(remote.Network(), remote.String())
}

// noLocalAddrRedialer redials with only remote addr.
type noLocalAddrRedialer struct{}

func (noLocalAddrRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	remote := oldConn.RemoteAddr()
	oldConn.Close() // ignore error

	dialer := net.Dialer{
		Control: greuse.Control,
	}
	return dialer.Dial(remote.Network(), remote.String())
}

// nopRedialer redials by doing nothing.
type nopRedialer struct{}

func (nopRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	return oldConn, nil
}
