package appinit

import (
	"errors"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
)

var (
	// Callback to prepare RxLoop for launching. If nil, RxLoop cannot be launched.
	// rxl has an assigned LCore with role iface.LCoreRole_RxLoop, but the callback may change it.
	// The callback must invoke rxl.SetCallback.
	BeforeStartRxl func(rxl *iface.RxLoop) (usr interface{}, e error)

	// Callback to cleanup after stopping RxLoop.
	AfterStopRxl func(rxl *iface.RxLoop, usr interface{})

	// Should this package automatically launch RxLoop?
	WantLaunchRxl bool = true

	// Should this package allocate/free RxLoop LCores?
	// If false, LCores with role iface.LCoreRole_RxLoop must be pre-allocated.
	WantAllocRxlLCore bool = true

	// Should this package allocate/free TxLoop LCores?
	// If false, LCores with role iface.LCoreRole_TxLoop must be pre-allocated.
	WantAllocTxlLCore bool = true
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

func (createfaceCallbacks) CreateRxMp(numaSocket dpdk.NumaSocket) (dpdk.PktmbufPool, error) {
	return MakePktmbufPool(MP_ETHRX, numaSocket), nil
}

type rxgUsr struct {
	rxl *iface.RxLoop
	usr interface{}
}

func (createfaceCallbacks) StartRxg(rxg iface.IRxGroup) (usr interface{}, e error) {
	if BeforeStartRxl == nil {
		return nil, errors.New("appinit.BeforeStartRxl is unset")
	}

	rxl := iface.NewRxLoop(rxg.GetNumaSocket())
	rxl.AddRxGroup(rxg)
	if WantAllocRxlLCore {
		rxl.SetLCore(dpdk.LCoreAlloc.Alloc(iface.LCoreRole_RxLoop, rxl.GetNumaSocket()))
	} else {
		rxl.SetLCore(dpdk.LCoreAlloc.Find(iface.LCoreRole_RxLoop, rxl.GetNumaSocket()))
	}

	defer func() {
		if e == nil {
			return
		}
		if lc := rxl.GetLCore(); WantAllocRxlLCore && lc != dpdk.LCORE_INVALID {
			dpdk.LCoreAlloc.Free(lc)
		}
		rxl.Close()
	}()

	var usr2 interface{}
	if usr2, e = BeforeStartRxl(rxl); e != nil {
		return nil, e
	}
	if WantLaunchRxl {
		if e = rxl.Launch(); e != nil {
			return nil, e
		}
	}
	return rxgUsr{rxl, usr2}, nil
}

func (createfaceCallbacks) StopRxg(rxg iface.IRxGroup, usr interface{}) {
	usr2 := usr.(rxgUsr)
	usr2.rxl.Stop()
	if AfterStopRxl != nil {
		AfterStopRxl(usr2.rxl, usr2.usr)
	}
	if WantAllocRxlLCore {
		dpdk.LCoreAlloc.Free(usr2.rxl.GetLCore())
	}
	usr2.rxl.Close()
}

func (createfaceCallbacks) StartTxl(txl *iface.TxLoop) (usr interface{}, e error) {
	if WantAllocTxlLCore {
		txl.SetLCore(dpdk.LCoreAlloc.Alloc(iface.LCoreRole_TxLoop, txl.GetNumaSocket()))
	} else {
		txl.SetLCore(dpdk.LCoreAlloc.Find(iface.LCoreRole_TxLoop, txl.GetNumaSocket()))
	}

	if e = txl.Launch(); e != nil && WantAllocTxlLCore {
		dpdk.LCoreAlloc.Free(txl.GetLCore())
	}
	return nil, e
}

func (createfaceCallbacks) StopTxl(txl *iface.TxLoop, usr interface{}) {
	txl.Stop()
	if WantAllocTxlLCore {
		dpdk.LCoreAlloc.Free(txl.GetLCore())
	}
}
