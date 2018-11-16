package appinit

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
)

var (
	// Callback to start RxLoop.
	// If nil, RxLoop cannot be started.
	StartRxl func(rxl iface.IRxLooper) (usr interface{}, e error)

	// Callback to stop RxLoop.
	StopRxl func(rxl iface.IRxLooper, usr interface{})

	// LCore reservation for TxLoop.
	// If nil, TxLoop can be started on any LCore.
	TxlLCoreReservation LCoreReservations
)

func EnableCreateFace(cfg createface.Config) error {
	return createface.Init(cfg, createfaceCallbacks{})
}

type createfaceCallbacks struct{}

func (createfaceCallbacks) CreateFaceMempools(numaSocket dpdk.NumaSocket) (mempools iface.Mempools, e error) {
	mempools.IndirectMp = MakePktmbufPool(MP_IND, numaSocket)
	mempools.NameMp = MakePktmbufPool(MP_NAME, numaSocket)
	mempools.HeaderMp = MakePktmbufPool(MP_HDR, numaSocket)
	return mempools, nil
}

func (createfaceCallbacks) CreateRxMp(mtu int, numaSocket dpdk.NumaSocket) (dpdk.PktmbufPool, error) {
	return MakePktmbufPool(MP_ETHRX, numaSocket), nil
}

func (createfaceCallbacks) StartRxl(rxl iface.IRxLooper) (usr interface{}, e error) {
	if StartRxl == nil {
		return nil, errors.New("appinit.NewRxThread is unset")
	}
	return StartRxl(rxl)
}

func (createfaceCallbacks) StopRxl(rxl iface.IRxLooper, usr interface{}) {
	if StopRxl != nil {
		StopRxl(rxl, usr)
	}
}

func (createfaceCallbacks) StartTxl(txl iface.ITxLooper) (usr interface{}, e error) {
	f := func() int {
		txl.TxLoop()
		return 0
	}
	if TxlLCoreReservation == nil {
		MustLaunch(f, txl.GetNumaSocket())
	} else {
		TxlLCoreReservation.MustReserve(txl.GetNumaSocket()).RemoteLaunch(f)
	}
	return nil, nil
}

func (createfaceCallbacks) StopTxl(txl iface.ITxLooper, usr interface{}) {
	txl.StopTxLoop()
}
