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
	nChanFaces = 0
	chanRxlUsr interface{}
	chanTxl    *iface.TxLoop
	chanTxlUsr interface{}
)

func startChanRxtx(face iface.IFace) (e error) {
	if nChanFaces == 0 {
		if chanRxlUsr, e = theCallbacks.StartRxg(iface.TheChanRxGroup); e != nil {
			return e
		}
		chanTxl = iface.NewTxLoop()
		if chanTxlUsr, e = theCallbacks.StartTxl(chanTxl); e != nil {
			return e
		}
	}

	chanTxl.AddFace(face)
	nChanFaces++
	return nil
}

func stopChanRxtx(face iface.IFace) {
	nChanFaces--
	chanTxl.RemoveFace(face)

	if nChanFaces == 0 {
		theCallbacks.StopRxg(iface.TheChanRxGroup, chanRxlUsr)
		chanRxlUsr = nil

		theCallbacks.StopTxl(chanTxl, chanTxlUsr)
		chanTxl.Close()
		chanTxl = nil
		chanTxlUsr = nil
	}
}
