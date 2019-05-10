package createface

import (
	"sync"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

var createDestroyLock sync.Mutex

func handleFaceNew(id iface.FaceId) {
	if !isInitialized {
		return
	}

	face := iface.Get(id)
	TheTxl.AddFace(face)
}

func handleFaceClosing(id iface.FaceId) {
	if !isInitialized {
		return
	}
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	face := iface.Get(id)
	TheTxl.RemoveFace(face)
}

func handleFaceClosed(id iface.FaceId) {
	if !isInitialized || id.GetKind() != iface.FaceKind_Eth {
		return
	}
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	for _, port := range ethface.ListPorts() {
		if len(port.ListFaces()) == 0 {
			port.Close()
		}
	}
}

func handleRxGroupAdd(rxg iface.IRxGroup) {
	if !isInitialized {
		return
	}

	TheRxl.AddRxGroup(rxg)
}

func handleRxGroupRemove(rxg iface.IRxGroup) {
	if !isInitialized {
		return
	}

	TheRxl.RemoveRxGroup(rxg)
}

var (
	theFaceNewEvt       = iface.OnFaceNew(handleFaceNew)
	theFaceClosingEvt   = iface.OnFaceClosing(handleFaceClosing)
	theFaceClosedEvt    = iface.OnFaceClosed(handleFaceClosed)
	theRxGroupAddEvt    = iface.OnRxGroupAdd(handleRxGroupAdd)
	theRxGroupRemoveEvt = iface.OnRxGroupRemove(handleRxGroupRemove)
)
