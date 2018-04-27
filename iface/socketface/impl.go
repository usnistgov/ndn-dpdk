package socketface

import (
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/faceuri"
)

type iImpl interface {
	// Receive packets on the socket and pass them to face.rxPkt.
	// Loop until a fatal error occurs or face.rxQuit receives a message.
	RxLoop(face *SocketFace)

	// Transmit one packet on the socket.
	Send(face *SocketFace, pkt dpdk.Packet) error

	// Return FaceUri describing an endpoint.
	FormatFaceUri(addr net.Addr) *faceuri.FaceUri

	// Redial the socket.
	Redial(oldConn net.Conn) (net.Conn, error)
}

var implByNetwork = make(map[string]iImpl)
