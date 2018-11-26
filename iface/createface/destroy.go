package createface

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

func handleFaceClosing(id iface.FaceId) {
	if !isInitialized {
		return
	}
	if kind := id.GetKind(); kind != iface.FaceKind_Mock && kind != iface.FaceKind_Socket {
		return
	}
	stopSmRxtx(iface.Get(id))
}

func handleFaceClosed(id iface.FaceId) {
	if !isInitialized || id.GetKind() != iface.FaceKind_Eth {
		return
	}
	for _, port := range ethface.ListPorts() {
		if port.CountFaces() == 0 {
			stopEthRxtx(port)
		}
	}
}

var (
	theFaceClosingEvt = iface.OnFaceClosing(handleFaceClosing)
	theFaceClosedEvt  = iface.OnFaceClosed(handleFaceClosed)
)
