package fetch

/*
#include "fetcher.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"math"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/pktqueue"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type FetcherConfig struct {
	RxQueue        pktqueue.Config
	WindowCapacity int
}

// Fetcher thread.
type Fetcher struct {
	dpdk.ThreadBase
	Id    uint8
	c     *C.Fetcher
	Logic *Logic
}

func New(id int, face iface.IFace, cfg FetcherConfig) (fetcher *Fetcher, e error) {
	if id < 0 || id > math.MaxUint8 {
		return nil, errors.New("id out of range")
	}
	faceId := face.GetFaceId()
	socket := face.GetNumaSocket()

	fetcher = new(Fetcher)
	fetcher.Id = uint8(id)
	fetcher.c = (*C.Fetcher)(dpdk.Zmalloc("Fetcher", C.sizeof_Fetcher, socket))
	fetcher.c.face = (C.FaceId)(faceId)
	fetcher.c.pitTokenBase = (C.uint64_t(id) << 56) | 0x66657463680000 // 'fetch\0\0'
	cfg.RxQueue.DisableCoDel = true
	if _, e := pktqueue.NewAt(unsafe.Pointer(&fetcher.c.rxQueue), cfg.RxQueue, fmt.Sprintf("Fetcher%d-%d_rxQ", faceId, id), socket); e != nil {
		dpdk.Free(fetcher.c)
		return nil, nil
	}
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
	return fetcher.SetTemplates([][]interface{}{{name}})
}

// Set multiple name prefixes and other InterestTemplate arguments.
// This is mainly useful for benchmarks.
func (fetcher *Fetcher) SetTemplates(tplArgsList [][]interface{}) error {
	if len(tplArgsList) < 1 || len(tplArgsList) > C.FETCHER_TEMPLATE_MAX {
		return fmt.Errorf("need between 1 and %d names", C.FETCHER_TEMPLATE_MAX)
	}
	for i, tplArgs := range tplArgsList {
		tpl := ndn.InterestTemplateFromPtr(unsafe.Pointer(&fetcher.c.tpl[i]))
		if e := tpl.Init(append([]interface{}{ndn.InterestMbufExtraHeadroom(appinit.SizeofEthLpHeaders())}, tplArgs...)...); e != nil {
			return e
		}

		if uintptr(tpl.PrefixL+1) >= unsafe.Sizeof(tpl.PrefixV) {
			return fmt.Errorf("names[%d] too long", i)
		}
		tpl.PrefixV[tpl.PrefixL] = uint8(ndn.TT_SegmentNameComponent)
		// put SegmentNameComponent TLV-TYPE in the buffer so that it's checked in same memcmp
	}
	fetcher.c.nTpls = C.uint8_t(len(tplArgsList))
	return nil
}

func (fetcher *Fetcher) GetRxQueue() pktqueue.PktQueue {
	return pktqueue.FromPtr(unsafe.Pointer(&fetcher.c.rxQueue))
}

func (fetcher *Fetcher) Launch() error {
	return fetcher.LaunchImpl(func() int {
		return int(C.Fetcher_Run(fetcher.c))
	})
}

func (fetcher *Fetcher) WaitForCompletion() error {
	return fetcher.StopImpl(dpdk.StopWait{})
}

func (fetcher *Fetcher) Stop() error {
	return fetcher.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&fetcher.c.stop)))
}

func (fetcher *Fetcher) Close() error {
	fetcher.Logic.Close()
	fetcher.GetRxQueue().Close()
	dpdk.Free(fetcher.c)
	return nil
}
