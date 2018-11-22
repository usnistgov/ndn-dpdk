package appinit

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
)

var (
	// Callback to prepare RxLoop for launching.
	// If nil, RxLoop cannot be launched.
	// The callback should perform rxl.SetLCore, rxl.SetCallback, and rxl.Launch.
	StartRxl func(rxl *iface.RxLoop) (usr interface{}, e error)

	// Callback to cleanup after stopping RxLoop.
	AfterStopRxl func(rxl *iface.RxLoop, usr interface{})

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

type rxgUsr struct {
	rxl *iface.RxLoop
	usr interface{}
}

func (createfaceCallbacks) StartRxg(rxg iface.IRxGroup) (usr interface{}, e error) {
	if StartRxl == nil {
		return nil, errors.New("appinit.StartRxl is unset")
	}

	var usr2 rxgUsr
	usr2.rxl = iface.NewRxLoop(rxg.GetNumaSocket())
	usr2.rxl.AddRxGroup(rxg)
	if usr2.usr, e = StartRxl(usr2.rxl); e != nil {
		usr2.rxl.Close()
		return nil, e
	}
	return usr2, nil
}

func (createfaceCallbacks) StopRxg(rxg iface.IRxGroup, usr interface{}) {
	usr2 := usr.(rxgUsr)
	usr2.rxl.Stop()
	if AfterStopRxl != nil {
		AfterStopRxl(usr2.rxl, usr2.usr)
	}
	usr2.rxl.Close()
}

func (createfaceCallbacks) StartTxl(txl *iface.TxLoop) (usr interface{}, e error) {
	lcr := TxlLCoreReservation
	if lcr == nil {
		lcr = NewLCoreReservations()
	}
	txl.SetLCore(lcr.MustReserve(txl.GetNumaSocket()))
	return nil, txl.Launch()
}

func (createfaceCallbacks) StopTxl(txl *iface.TxLoop, usr interface{}) {
	txl.Stop()
}
