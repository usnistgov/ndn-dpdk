package createface

import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

type ethRxtx struct {
	rxgUsr interface{}
	txl    *iface.TxLoop
	txlUsr interface{}
	nFaces int
}

var ethRxtxByPort = make(map[dpdk.EthDev]ethRxtx)

func ethRxgFromPort(port *ethface.Port) iface.IRxGroup {
	rxgs := port.ListRxGroups()
	if len(rxgs) != 1 {
		panic("unexpected len(Port.ListRxGroups())")
	}
	return rxgs[0]
}

func startEthRxtx(port *ethface.Port) (e error) {
	var rxtx ethRxtx

	rxg := ethRxgFromPort(port)
	if rxtx.rxgUsr, e = theCallbacks.StartRxg(rxg); e != nil {
		return e
	}

	var faces []iface.IFace
	if port.GetMulticastFace() != nil {
		faces = append(faces, port.GetMulticastFace())
	}
	for _, face := range port.ListUnicastFaces() {
		faces = append(faces, face)
	}
	for _, face := range faces {
		face.EnableThreadSafeTx(theConfig.EthTxqPkts)
	}

	rxtx.txl = iface.NewTxLoop(faces...)
	if rxtx.txlUsr, e = theCallbacks.StartTxl(rxtx.txl); e != nil {
		return e
	}

	rxtx.nFaces = len(faces)
	ethRxtxByPort[port.GetEthDev()] = rxtx
	return nil
}

func stopEthRxtx(face *ethface.EthFace) {
	port := face.GetPort()
	ethdev := port.GetEthDev()
	rxtx := ethRxtxByPort[ethdev]
	rxtx.nFaces--
	if rxtx.nFaces > 0 {
		ethRxtxByPort[ethdev] = rxtx
		return
	}

	theCallbacks.StopRxg(ethRxgFromPort(port), rxtx.rxgUsr)
	theCallbacks.StopTxl(rxtx.txl, rxtx.txlUsr)
	delete(ethRxtxByPort, ethdev)
}

var (
	nSmFaces = 0
	smRxlUsr interface{}
	smTxl    *iface.TxLoop
	smTxlUsr interface{}
)

func startSmRxtx(face iface.IFace) (e error) {
	if nSmFaces == 0 {
		if smRxlUsr, e = theCallbacks.StartRxg(iface.TheChanRxGroup); e != nil {
			return e
		}
		smTxl = iface.NewTxLoop()
		if smTxlUsr, e = theCallbacks.StartTxl(smTxl); e != nil {
			return e
		}
	}

	switch face.GetFaceId().GetKind() {
	case iface.FaceKind_Mock:
		face.EnableThreadSafeTx(theConfig.MockTxqPkts)
	case iface.FaceKind_Socket:
		face.EnableThreadSafeTx(theConfig.SockTxqPkts)
	}
	smTxl.AddFace(face)
	nSmFaces++

	return nil
}

func stopSmRxtx(face iface.IFace) {
	nSmFaces--
	smTxl.RemoveFace(face)

	if nSmFaces == 0 {
		theCallbacks.StopRxg(iface.TheChanRxGroup, smRxlUsr)
		smRxlUsr = nil

		theCallbacks.StopTxl(smTxl, smTxlUsr)
		smTxl.Close()
		smTxl = nil
		smTxlUsr = nil
	}
}
