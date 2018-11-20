package createface

import (
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/mockface"
	"ndn-dpdk/iface/socketface"
)

type ethRxtx struct {
	rxl    *ethface.RxLoop
	rxlUsr interface{}
	txl    *iface.TxLoop
	txlUsr interface{}
}

var ethRxtxByPort = make(map[dpdk.EthDev]ethRxtx)

func startEthRxtx(port *ethface.Port) (e error) {
	numaSocket := port.GetNumaSocket()
	var rxtx ethRxtx

	rxtx.rxl = ethface.NewRxLoop(1, numaSocket)
	if e = rxtx.rxl.AddPort(port); e != nil {
		return e
	}
	if rxtx.rxlUsr, e = theCallbacks.StartRxl(rxtx.rxl); e != nil {
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

	ethRxtxByPort[port.GetEthDev()] = rxtx
	return nil
}

func stopEthRxtx(port *ethface.Port) {
	ethdev := port.GetEthDev()
	rxtx := ethRxtxByPort[ethdev]
	theCallbacks.StopRxl(rxtx.rxl, rxtx.rxlUsr)
	theCallbacks.StopTxl(rxtx.txl, rxtx.txlUsr)
	delete(ethRxtxByPort, ethdev)
}

var (
	nSockActive    = 0
	nMockActive    = 0
	sockRxl        *socketface.RxGroup
	sockRxlUsr     interface{}
	mockRxlUsr     interface{}
	sockMockTxl    *iface.TxLoop
	sockMockTxlUsr interface{}
)

func startSockMockTxl() (e error) {
	sockMockTxl = iface.NewTxLoop()
	sockMockTxlUsr, e = theCallbacks.StartTxl(sockMockTxl)
	return e
}

func stopSockMockTxl() {
	theCallbacks.StopTxl(sockMockTxl, sockMockTxlUsr)
	sockMockTxl = nil
	sockMockTxlUsr = nil
}

func startSockRxtx(face *socketface.SocketFace) (e error) {
	if nSockActive == 0 {
		sockRxl = socketface.NewRxGroup()
		if sockRxlUsr, e = theCallbacks.StartRxl(sockRxl); e != nil {
			return e
		}
	}
	if nSockActive+nMockActive == 0 {
		if e = startSockMockTxl(); e != nil {
			return e
		}
	}

	face.EnableThreadSafeTx(theConfig.SockTxqPkts)
	sockRxl.AddFace(face)
	sockMockTxl.AddFace(face)
	nSockActive++
	return nil
}

func stopSockRxtx(face *socketface.SocketFace) {
	sockRxl.RemoveFace(face)
	sockMockTxl.RemoveFace(face)
	nSockActive--

	if nSockActive == 0 {
		theCallbacks.StopRxl(sockRxl, sockRxlUsr)
		sockRxl = nil
		sockRxlUsr = nil
	}
	if nSockActive+nMockActive == 0 {
		stopSockMockTxl()
	}
}

func startMockRxtx(face *mockface.MockFace) (e error) {
	if nMockActive == 0 {
		if mockRxlUsr, e = theCallbacks.StartRxl(mockface.TheRxLoop); e != nil {
			return e
		}
	}
	if nSockActive+nMockActive == 0 {
		if e = startSockMockTxl(); e != nil {
			return e
		}
	}

	face.EnableThreadSafeTx(theConfig.MockTxqPkts)
	sockMockTxl.AddFace(face)
	nMockActive++
	return nil
}

func stopMockRxtx(face *mockface.MockFace) {
	sockMockTxl.RemoveFace(face)
	nMockActive--

	if nMockActive == 0 {
		theCallbacks.StopRxl(mockface.TheRxLoop, mockRxlUsr)
		mockRxlUsr = nil
	}
	if nSockActive+nMockActive == 0 {
		stopSockMockTxl()
	}
}
