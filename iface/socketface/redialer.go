package socketface

import (
	"net"
)

// Provides a Redial function that reuses LocalAddr.
type localAddrRedialer struct{}

func (localAddrRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	local, remote := oldConn.LocalAddr(), oldConn.RemoteAddr()
	oldConn.Close()
	dialer := net.Dialer{LocalAddr: local}
	return dialer.Dial(remote.Network(), remote.String())
}

// Provides a Redial function that does not reuse LocalAddr.
type noLocalAddrRedialer struct{}

func (noLocalAddrRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	remote := oldConn.RemoteAddr()
	oldConn.Close()
	return net.Dial(remote.Network(), remote.String())
}

// Provides a Redial function that does nothing.
type nopRedialer struct{}

func (nopRedialer) Redial(oldConn net.Conn) (net.Conn, error) {
	return oldConn, nil
}
