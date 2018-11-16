package createface

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
)

func handleFaceClosing(id iface.FaceId) {
	if !isInitialized {
		return
	}
	face := iface.Get(id)
	switch id.GetKind() {
	case iface.FaceKind_Mock:
		stopMockRxtx(face.(*mockface.MockFace))
	case iface.FaceKind_Eth:
		stopEthRxtx(face.(*ethface.EthFace).GetPort())
	case iface.FaceKind_Socket:
		stopSockRxtx(face.(*socketface.SocketFace))
	}
}

var theFaceClosingEvt = iface.OnFaceClosing(handleFaceClosing)
