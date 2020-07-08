package createface

import (
	"sync"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethface"
)

var createDestroyLock sync.Mutex

func handleFaceClosed(id iface.ID) {
	createDestroyLock.Lock()
	defer createDestroyLock.Unlock()

	for _, port := range ethface.ListPorts() {
		if port.CountFaces() == 0 {
			port.Close()
		}
	}
}

var theFaceClosedEvt = iface.OnFaceClosed(handleFaceClosed)
