package fetch

/*
#include "../../csrc/fetch/fetcher.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/ping/pingmempool"
	"github.com/usnistgov/ndn-dpdk/container/pktqueue"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type FetcherConfig struct {
	NThreads       int
	NProcs         int
	RxQueue        pktqueue.Config
	WindowCapacity int
}

// Fetcher controls fetch threads and fetch procedures on a face.
type Fetcher struct {
	fth          []*fetchThread
	fp           []*C.FetchProc
	nActiveProcs int
}

func New(face iface.Face, cfg FetcherConfig) (fetcher *Fetcher, e error) {
	if cfg.NThreads == 0 {
		cfg.NThreads = 1
	}
	if cfg.NProcs == 0 {
		cfg.NProcs = 1
	}
	cfg.RxQueue.DisableCoDel = true

	faceID := face.ID()
	socket := face.NumaSocket()
	interestMp := (*C.struct_rte_mempool)(pingmempool.Interest.MakePool(socket).GetPtr())

	fetcher = new(Fetcher)
	fetcher.fth = make([]*fetchThread, cfg.NThreads)
	for i := range fetcher.fth {
		fth := new(fetchThread)
		fth.c = (*C.FetchThread)(eal.Zmalloc("FetchThread", C.sizeof_FetchThread, socket))
		fth.c.face = (C.FaceID)(faceID)
		fth.c.interestMp = interestMp
		C.NonceGen_Init(&fth.c.nonceGen)
		eal.InitStopFlag(unsafe.Pointer(&fth.c.stop))
		fetcher.fth[i] = fth
	}

	fetcher.fp = make([]*C.FetchProc, cfg.NProcs)
	for i := range fetcher.fp {
		fp := (*C.FetchProc)(eal.Zmalloc("FetchProc", C.sizeof_FetchProc, socket))
		if _, e := pktqueue.NewAt(unsafe.Pointer(&fp.rxQueue), cfg.RxQueue, fmt.Sprintf("Fetcher%d-%d_rxQ", faceID, i), socket); e != nil {
			return nil, e
		}
		fp.pitToken = (C.uint64_t(i) << 56) | 0x6665746368 // 'fetch'
		fetcher.fp[i] = fp
		fetcher.GetLogic(i).Init(cfg.WindowCapacity, socket)
	}

	return fetcher, nil
}

func (fetcher *Fetcher) GetFace() iface.Face {
	return iface.Get(iface.ID(fetcher.fth[0].c.face))
}

func (fetcher *Fetcher) CountThreads() int {
	return len(fetcher.fth)
}

func (fetcher *Fetcher) GetThread(i int) eal.IThread {
	return fetcher.fth[i]
}

func (fetcher *Fetcher) CountProcs() int {
	return len(fetcher.fp)
}

func (fetcher *Fetcher) GetRxQueue(i int) *pktqueue.PktQueue {
	return pktqueue.FromPtr(unsafe.Pointer(&fetcher.fp[i].rxQueue))
}

func (fetcher *Fetcher) GetLogic(i int) *Logic {
	return LogicFromPtr(unsafe.Pointer(&fetcher.fp[i].logic))
}

func (fetcher *Fetcher) Reset() {
	for _, fth := range fetcher.fth {
		fth.c.head.next = nil
	}
	for i := range fetcher.fp {
		fetcher.GetLogic(i).Reset()
	}
	fetcher.nActiveProcs = 0
}

// Set name prefix and other InterestTemplate arguments.
func (fetcher *Fetcher) AddTemplate(tplArgs ...interface{}) (i int, e error) {
	i = fetcher.nActiveProcs
	if i >= len(fetcher.fp) {
		return -1, errors.New("too many prefixes")
	}

	fp := fetcher.fp[i]
	tpl := ndni.InterestTemplateFromPtr(unsafe.Pointer(&fp.tpl))
	if e := tpl.Init(tplArgs...); e != nil {
		return -1, e
	}

	if uintptr(tpl.PrefixL+1) >= unsafe.Sizeof(tpl.PrefixV) {
		return -1, errors.New("name too long")
	}
	tpl.PrefixV[tpl.PrefixL] = uint8(an.TtSegmentNameComponent)
	// put SegmentNameComponent TLV-TYPE in the buffer so that it's checked in same memcmp

	rs := urcu.NewReadSide()
	defer rs.Close()
	fth := fetcher.fth[i%len(fetcher.fth)]
	C.cds_hlist_add_head_rcu(&fp.fthNode, &fth.c.head)
	fetcher.nActiveProcs++
	return i, nil
}

func (fetcher *Fetcher) Launch() {
	for _, fth := range fetcher.fth {
		fth.Launch()
	}
}

func (fetcher *Fetcher) Stop() {
	for _, fth := range fetcher.fth {
		fth.Stop()
	}
}

func (fetcher *Fetcher) Close() error {
	for i, fp := range fetcher.fp {
		fetcher.GetRxQueue(i).Close()
		fetcher.GetLogic(i).Close()
		eal.Free(fp)
	}
	for _, fth := range fetcher.fth {
		fth.Close()
	}
	return nil
}

type fetchThread struct {
	eal.ThreadBase
	c *C.FetchThread
}

func (fth *fetchThread) Launch() error {
	return fth.LaunchImpl(func() int {
		return int(C.FetchThread_Run(fth.c))
	})
}

func (fth *fetchThread) Stop() error {
	return fth.StopImpl(eal.NewStopFlag(unsafe.Pointer(&fth.c.stop)))
}

func (fth *fetchThread) Close() error {
	eal.Free(fth.c)
	return nil
}
