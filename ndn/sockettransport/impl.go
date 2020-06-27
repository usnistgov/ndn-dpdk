package sockettransport

import (
	"net"
)

type impl interface {
	// Dial the socket.
	Dial(network, local, remote string) (net.Conn, error)

	// Redial the socket.
	Redial(oldConn net.Conn) (net.Conn, error)

	// Receive packets on the socket and pass them to tr.rx.
	RxLoop(tr *Transport)
}

var implByNetwork = make(map[string]impl)

// noLocalAddrDialer dials with only remote addr.
type noLocalAddrDialer struct{}

func (noLocalAddrDialer) Dial(network, local, remote string) (net.Conn, error) {
	return net.Dial(network, remote)
}

// localAddrRedialer redials reusing local addr.
type localAddrRedialer struct{}

func (localAddrRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	local, remote := oldConn.LocalAddr(), oldConn.RemoteAddr()
	oldConn.Close()
	dialer := net.Dialer{LocalAddr: local}
	return dialer.Dial(remote.Network(), remote.String())
}

// noLocalAddrRedialer redials with only remote addr.
type noLocalAddrRedialer struct{}

func (noLocalAddrRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	remote := oldConn.RemoteAddr()
	oldConn.Close()
	return net.Dial(remote.Network(), remote.String())
}

// nopRedialer redials doing thing.
type nopRedialer struct{}

func (nopRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	return oldConn, nil
}
