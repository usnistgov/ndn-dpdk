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

	createfaceConfig createface.Config
)

func EnableCreateFace() error {
	if BeforeStartRxl == nil {
		return errors.New("appinit.BeforeStartRxl is unset")
	}

	rxl := iface.NewRxLoop(dpdk.NUMA_SOCKET_ANY)
	if WantAllocRxlLCore {
		rxl.SetLCore(dpdk.LCoreAlloc.Alloc(iface.LCoreRole_RxLoop, dpdk.NUMA_SOCKET_ANY))
	} else {
		rxl.SetLCore(dpdk.LCoreAlloc.Find(iface.LCoreRole_RxLoop, dpdk.NUMA_SOCKET_ANY))
	}
	if _, e := BeforeStartRxl(rxl); e != nil {
		return e
	}
	if WantLaunchRxl {
		if e := rxl.Launch(); e != nil {
			return e
		}
	}

	txl := iface.NewTxLoop()
	if WantAllocTxlLCore {
		txl.SetLCore(dpdk.LCoreAlloc.Alloc(iface.LCoreRole_TxLoop, txl.GetNumaSocket()))
	} else {
		txl.SetLCore(dpdk.LCoreAlloc.Find(iface.LCoreRole_TxLoop, txl.GetNumaSocket()))
	}
	if e := txl.Launch(); e != nil {
		if WantAllocTxlLCore {
			dpdk.LCoreAlloc.Free(txl.GetLCore())
		}
		return e
	}

	createface.TheRxl = rxl
	createface.TheTxl = txl
	return createface.Init(createfaceConfig, createfaceCallbacks{})
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
