package createface

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

func handleFaceClosing(id iface.FaceId) {
	if !isInitialized {
		return
	}
	face := iface.Get(id)
	switch id.GetKind() {
	case iface.FaceKind_Eth:
		stopEthRxtx(face.(*ethface.EthFace))
	case iface.FaceKind_Mock, iface.FaceKind_Socket:
		stopSmRxtx(face)
	}
}

var theFaceClosingEvt = iface.OnFaceClosing(handleFaceClosing)
