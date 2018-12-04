package createface

import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

type ethRxtx struct {
	rxgUsrs map[unsafe.Pointer]interface{}
	txl     *iface.TxLoop
	txlUsr  interface{}
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

	rxtx.rxgUsrs = make(map[unsafe.Pointer]interface{})
	for _, rxg := range port.ListRxGroups() {
		rxgUsr, e := theCallbacks.StartRxg(rxg)
		if e != nil {
			return e
		}
		rxtx.rxgUsrs[rxg.GetPtr()] = rxgUsr
	}

	var faces []iface.IFace
	if face := port.GetMulticastFace(); face != nil {
		faces = append(faces, face)
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

	ethRxtxByPort[port.GetEthDev()] = rxtx
	return nil
}

func stopEthFaceRxtx(face *ethface.EthFace) {
	port := face.GetPort()
	ethdev := port.GetEthDev()
	rxtx := ethRxtxByPort[ethdev]

	for _, rxg := range face.ListRxGroups() {
		if _, ok := rxg.(*ethface.RxFlow); !ok {
			continue
		}
		rxgUsr := rxtx.rxgUsrs[rxg.GetPtr()]
		theCallbacks.StopRxg(rxg, rxgUsr)
	}
	rxtx.txl.RemoveFace(face)
}

func stopEthPortRxtx(port *ethface.Port) {
	ethdev := port.GetEthDev()
	rxtx := ethRxtxByPort[ethdev]

	for _, rxg := range port.ListRxGroups() {
		if _, ok := rxg.(*ethface.RxTable); !ok {
			continue
		}
		rxgUsr := rxtx.rxgUsrs[rxg.GetPtr()]
		theCallbacks.StopRxg(rxg, rxgUsr)
	}
	theCallbacks.StopTxl(rxtx.txl, rxtx.txlUsr)
	port.Close()
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
