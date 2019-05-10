package createface

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

func startEthRxtx(port *ethface.Port) (e error) {
	for _, rxg := range port.ListRxGroups() {
		TheRxl.AddRxGroup(rxg)
	}

	if face := port.GetMulticastFace(); face != nil {
		TheTxl.AddFace(face)
	}
	for _, face := range port.ListUnicastFaces() {
		TheTxl.AddFace(face)
	}

	return nil
}

func stopEthFaceRxtx(face *ethface.EthFace) {
	for _, rxg := range face.ListRxGroups() {
		if _, ok := rxg.(*ethface.RxFlow); !ok {
			continue
		}
		TheRxl.RemoveRxGroup(rxg)
	}
	TheTxl.RemoveFace(face)
}

func stopEthPortRxtx(port *ethface.Port) {
	for _, rxg := range port.ListRxGroups() {
		if _, ok := rxg.(*ethface.RxTable); !ok {
			continue
		}
		TheRxl.RemoveRxGroup(rxg)
	}
	port.Close()
}

var (
	nChanFaces = 0
)

func startChanRxtx(face iface.IFace) (e error) {
	if nChanFaces == 0 {
		TheRxl.AddRxGroup(iface.TheChanRxGroup)
	}
	TheTxl.AddFace(face)
	nChanFaces++
	return nil
}

func stopChanRxtx(face iface.IFace) {
	nChanFaces--
	TheTxl.RemoveFace(face)
	if nChanFaces == 0 {
		TheRxl.RemoveRxGroup(iface.TheChanRxGroup)
	}
}
