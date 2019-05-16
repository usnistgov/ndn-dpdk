package createface

import (
	"math"
	"sync"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

var createDestroyLock sync.Mutex

func handleFaceNew(id iface.FaceId) {
	if theConfig.Disabled {
		return
	}
	// lock held by Create()

	face := iface.Get(id)

	var bestTxl *iface.TxLoop
	bestScore := math.MaxInt32
	for _, txl := range theTxls {
		score := len(txl.ListFaces())
		if !txl.GetNumaSocket().Match(face.GetNumaSocket()) {
			score += 1000000
		}

		if score <= bestScore {
			bestTxl = txl
			bestScore = score
		}
	}

	bestTxl.AddFace(face)
}

func handleFaceClosing(id iface.FaceId) {
	if theConfig.Disabled {
		return
	}
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	face := iface.Get(id)
	for _, txl := range theTxls {
		txl.RemoveFace(face)
	}
}

func handleFaceClosed(id iface.FaceId) {
	if theConfig.Disabled {
		return
	}
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	if id.GetKind() == iface.FaceKind_Eth {
		for _, port := range ethface.ListPorts() {
			if len(port.ListFaces()) == 0 {
				port.Close()
			}
		}
	}
}

func handleRxGroupAdd(rxg iface.IRxGroup) {
	if theConfig.Disabled {
		return
	}
	// lock held by Create()

	var bestRxl *iface.RxLoop
	bestScore := math.MaxInt32
	for _, rxl := range theRxls {
		score := 1000*len(rxl.ListRxGroups()) + len(rxl.ListFaces())
		if !rxl.GetNumaSocket().Match(rxg.GetNumaSocket()) {
			score += 1000000
		}

		if score <= bestScore {
			bestRxl = rxl
			bestScore = score
		}
	}

	bestRxl.AddRxGroup(rxg)
}

func handleRxGroupRemove(rxg iface.IRxGroup) {
	if theConfig.Disabled {
		return
	}
	// lock held by Create() or handleFaceClosed()

	rxg.GetRxLoop().RemoveRxGroup(rxg)
}

var (
	theFaceNewEvt       = iface.OnFaceNew(handleFaceNew)
	theFaceClosingEvt   = iface.OnFaceClosing(handleFaceClosing)
	theFaceClosedEvt    = iface.OnFaceClosed(handleFaceClosed)
	theRxGroupAddEvt    = iface.OnRxGroupAdd(handleRxGroupAdd)
	theRxGroupRemoveEvt = iface.OnRxGroupRemove(handleRxGroupRemove)
)
