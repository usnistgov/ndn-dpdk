package socketface

import (
	"errors"
	"net"
)

// Provides a ValidateAddr function for unix and unixgram schemes.
type unixAddrValidator struct{}

func (unixAddrValidator) ValidateAddr(network, address string, isLocal bool) (e error) {
	if isLocal {
		if address != "" && address != "@" {
			return errors.New("must be empty or '@'")
		}
		return nil
	}
	_, e = net.ResolveUnixAddr(network, address)
	return e
}

// Provides a Dial function that only uses remote addr.
type noLocalAddrDialer struct{}

func (noLocalAddrDialer) Dial(network, local, remote string) (net.Conn, error) {
	return net.Dial(network, remote)
}

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
