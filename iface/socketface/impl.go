package socketface

import (
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/faceuri"
)

type iImpl interface {
	// Validate address in Locator.
	ValidateAddr(network, address string, isLocal bool) error

	// Return FaceUri describing an endpoint.
	FormatFaceUri(addr net.Addr) *faceuri.FaceUri

	// Dial the socket.
	Dial(network, local, remote string) (net.Conn, error)

	// Redial the socket.
	Redial(oldConn net.Conn) (net.Conn, error)

	// Receive packets on the socket and pass them to face.rxPkt.
	// Loop until a fatal error occurs or face.rxQuit receives a message.
	RxLoop(face *SocketFace)

	// Transmit one packet on the socket.
	Send(face *SocketFace, pkt dpdk.Packet) error
}

var implByNetwork = make(map[string]iImpl)
