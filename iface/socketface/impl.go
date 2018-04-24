package socketface

import (
	"net"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/faceuri"
)

type iImpl interface {
	// Receive packets on the socket and post them to face.rxQueue.
	// Loop until a fatal error occurs or face.rxQuit receives a message.
	// Increment face.rxCongestions when a packet arrives but face.rxQueue is full.
	RxLoop(face *SocketFace)

	// Transmit one packet on the socket.
	Send(face *SocketFace, pkt dpdk.Packet) error

	// Return FaceUri describing an endpoint.
	FormatFaceUri(addr net.Addr) *faceuri.FaceUri
}

var implByNetwork = make(map[string]iImpl)
