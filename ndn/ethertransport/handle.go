//go:build linux

package ethertransport

import (
	"fmt"
	"io"
	"net"
	"reflect"
	"syscall"

	"github.com/mdlayher/packet"
	"github.com/mdlayher/socket"
	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"golang.org/x/sys/unix"
)

// ConnHandle represents an AF_PACKET socket.
// It enhances *packet.Conn with io.ReadWriteCloser compatibility and multicast membership functions.
type ConnHandle struct {
	*packet.Conn
	netif *net.Interface
}

var (
	_ io.ReadWriteCloser = (*ConnHandle)(nil)
	_ syscall.Conn       = (*ConnHandle)(nil)
)

// Read implements io.Reader interface.
func (h *ConnHandle) Read(buf []byte) (n int, e error) {
	n, _, e = h.Conn.ReadFrom(buf)
	return
}

// Write implements io.Writer interface.
func (h *ConnHandle) Write(pkt []byte) (n int, e error) {
	if len(pkt) < 14 {
		return 0, io.ErrShortBuffer
	}
	return h.Conn.WriteTo(pkt, &packet.Addr{
		HardwareAddr: pkt[0:6],
	})
}

// SocketConn access the underlying *socket.Conn instance.
func (h *ConnHandle) SocketConn() *socket.Conn {
	return (*socket.Conn)(reflect.ValueOf(h.Conn).Elem().FieldByName("c").UnsafePointer())
}

// JoinMulticast joins a multicast group.
func (h *ConnHandle) JoinMulticast(group net.HardwareAddr) (leave func() error, e error) {
	if !macaddr.IsMulticast(group) {
		return nil, macaddr.ErrMulticast
	}

	mreq := unix.PacketMreq{
		Ifindex: int32(h.netif.Index),
		Type:    unix.PACKET_MR_MULTICAST,
	}
	mreq.Alen = uint16(copy(mreq.Address[:], group))

	leave = func() error {
		return h.SocketConn().SetsockoptPacketMreq(unix.SOL_PACKET, unix.PACKET_DROP_MEMBERSHIP, &mreq)
	}
	e = h.SocketConn().SetsockoptPacketMreq(unix.SOL_PACKET, unix.PACKET_ADD_MEMBERSHIP, &mreq)
	return
}

// NewConnHandle opens ConnHandle on a network interface.
func NewConnHandle(netif *net.Interface, protocol int) (*ConnHandle, error) {
	if netif.Flags&net.FlagUp == 0 {
		return nil, fmt.Errorf("netif %s is not UP", netif.Name)
	}

	if protocol == 0 {
		protocol = unix.ETH_P_ALL
	}
	conn, e := packet.Listen(netif, packet.Raw, protocol, nil)
	if e != nil {
		return nil, fmt.Errorf("packet.Listen(%s,RAW,EthernetTypeNDN): %w", netif.Name, e)
	}

	return &ConnHandle{conn, netif}, nil
}
