package createface

import (
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

func startEthRxtx(face *ethface.EthFace) (e error) {
	for _, rxg := range face.ListRxGroups() {
		TheRxl.AddRxGroup(rxg)
	}
	TheTxl.AddFace(face)
	return nil
}

func stopEthFaceRxtx(face *ethface.EthFace) {
	for _, rxg := range face.ListRxGroups() {
		TheRxl.RemoveRxGroup(rxg)
	}
	TheTxl.RemoveFace(face)
}

func stopEthPortRxtx(port *ethface.Port) {
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
