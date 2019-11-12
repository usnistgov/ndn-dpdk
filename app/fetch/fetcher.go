package fetch

/*
#include "fetcher.h"
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type FetcherConfig struct {
	WindowCapacity int
}

// Fetcher thread.
type Fetcher struct {
	dpdk.ThreadBase
	c     *C.Fetcher
	Logic *Logic
}

func New(face iface.IFace, cfg FetcherConfig) (fetcher *Fetcher, e error) {
	socket := face.GetNumaSocket()

	fetcher = new(Fetcher)
	fetcher.c = (*C.Fetcher)(dpdk.Zmalloc("Fetcher", C.sizeof_Fetcher, socket))
	fetcher.c.face = (C.FaceId)(face.GetFaceId())
	fetcher.c.interestMbufHeadroom = C.uint16_t(appinit.SizeofEthLpHeaders() + ndn.EncodeInterest_GetHeadroom())
	fetcher.c.interestMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(appinit.MP_INT, socket).GetPtr())
	C.NonceGen_Init(&fetcher.c.nonceGen)

	fetcher.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&fetcher.c.stop))

	fetcher.Logic = LogicFromPtr(unsafe.Pointer(&fetcher.c.logic))
	fetcher.Logic.Init(cfg.WindowCapacity, socket)

	return fetcher, nil
}

func (fetcher *Fetcher) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(fetcher.c.face))
}

func (fetcher *Fetcher) SetName(name *ndn.Name) error {
	tpl := ndn.NewInterestTemplate()
	tpl.SetNamePrefix(name)
	return tpl.CopyToC(unsafe.Pointer(&fetcher.c.tpl),
		unsafe.Pointer(&fetcher.c.tplPrepareBuffer), unsafe.Sizeof(fetcher.c.tplPrepareBuffer),
		unsafe.Pointer(&fetcher.c.prefixBuffer), unsafe.Sizeof(fetcher.c.prefixBuffer))
}

func (fetcher *Fetcher) SetRxQueue(queue dpdk.Ring) {
	fetcher.c.rxQueue = (*C.struct_rte_ring)(queue.GetPtr())
}

func (fetcher *Fetcher) Launch() error {
	return fetcher.LaunchImpl(func() int {
		return int(C.Fetcher_Run(fetcher.c))
	})
}

func (fetcher *Fetcher) Stop() error {
	return fetcher.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&fetcher.c.stop)))
}

func (fetcher *Fetcher) Close() error {
	fetcher.Logic.Close()
	dpdk.Free(fetcher.c)
	return nil
}
