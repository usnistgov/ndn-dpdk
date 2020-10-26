// Package fetch simulates bulk file transfer traffic patterns.
package fetch

/*
#include "../../csrc/fetch/fetcher.h"
*/
import "C"
import (
	"errors"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

var fetcherByFace = make(map[iface.ID]*Fetcher)

// FetcherConfig contains Fetcher configuration.
type FetcherConfig struct {
	NThreads       int                  `json:"nThreads,omitempty"`
	NProcs         int                  `json:"nProcs,omitempty"`
	RxQueue        iface.PktQueueConfig `json:"rxQueue,omitempty"`
	WindowCapacity int                  `json:"windowCapacity,omitempty"`
}

// Fetcher controls fetch threads and fetch procedures on a face.
// A fetch procedure retrieves Data under a single name prefix, and has independent congestion control.
// A fetch thread runs on an lcore, and can serve multiple fetch procedures.
type Fetcher struct {
	fth          []*fetchThread
	fp           []*C.FetchProc
	nActiveProcs int
}

// New creates a Fetcher.
func New(face iface.Face, cfg FetcherConfig) (*Fetcher, error) {
	if cfg.NThreads == 0 {
		cfg.NThreads = 1
	}
	if cfg.NProcs == 0 {
		cfg.NProcs = 1
	}
	cfg.RxQueue.DisableCoDel = true

	faceID := face.ID()
	socket := face.NumaSocket()
	interestMp := (*C.struct_rte_mempool)(ndni.InterestMempool.MakePool(socket).Ptr())

	fetcher := &Fetcher{
		fth: make([]*fetchThread, cfg.NThreads),
		fp:  make([]*C.FetchProc, cfg.NProcs),
	}
	for i := range fetcher.fth {
		fth := &fetchThread{
			c: (*C.FetchThread)(eal.Zmalloc("FetchThread", C.sizeof_FetchThread, socket)),
		}
		fth.c.face = (C.FaceID)(faceID)
		fth.c.interestMp = interestMp
		C.NonceGen_Init(&fth.c.nonceGen)
		fth.Thread = ealthread.New(
			cptr.Func0.C(unsafe.Pointer(C.FetchThread_Run), unsafe.Pointer(fth.c)),
			ealthread.InitStopFlag(unsafe.Pointer(&fth.c.stop)),
		)
		fetcher.fth[i] = fth
	}

	for i := range fetcher.fp {
		fp := (*C.FetchProc)(eal.Zmalloc("FetchProc", C.sizeof_FetchProc, socket))
		if e := iface.PktQueueFromPtr(unsafe.Pointer(&fp.rxQueue)).Init(cfg.RxQueue, socket); e != nil {
			return nil, e
		}
		fp.pitToken = (C.uint64_t(i) << 56) | 0x6665746368 // 'fetch'
		fetcher.fp[i] = fp
		fetcher.Logic(i).Init(cfg.WindowCapacity, socket)
	}

	fetcherByFace[faceID] = fetcher
	return fetcher, nil
}

// Face returns the face.
func (fetcher *Fetcher) Face() iface.Face {
	return iface.Get(iface.ID(fetcher.fth[0].c.face))
}

// CountThreads returns number of threads.
func (fetcher *Fetcher) CountThreads() int {
	return len(fetcher.fth)
}

// Thread returns i-th thread.
func (fetcher *Fetcher) Thread(i int) ealthread.Thread {
	return fetcher.fth[i]
}

// CountProcs returns number of fetch procedures.
func (fetcher *Fetcher) CountProcs() int {
	return len(fetcher.fp)
}

// RxQueue returns the RX queue of i-th fetch procedure.
func (fetcher *Fetcher) RxQueue(i int) *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&fetcher.fp[i].rxQueue))
}

// Logic returns the Logic of i-th fetch procedure.
func (fetcher *Fetcher) Logic(i int) *Logic {
	return LogicFromPtr(unsafe.Pointer(&fetcher.fp[i].logic))
}

// Reset resets all Logics.
// If the fetcher is running, it is automatically stopped.
func (fetcher *Fetcher) Reset() {
	fetcher.Stop()
	for _, fth := range fetcher.fth {
		fth.c.head.next = nil
	}
	for i := range fetcher.fp {
		fetcher.Logic(i).Reset()
	}
	fetcher.nActiveProcs = 0
}

// AddTemplate sets name prefix and other InterestTemplate arguments.
// Return index of fetch procedure.
func (fetcher *Fetcher) AddTemplate(tplArgs ...interface{}) (i int, e error) {
	i = fetcher.nActiveProcs
	if i >= len(fetcher.fp) {
		return -1, errors.New("too many templates")
	}

	fp := fetcher.fp[i]
	tpl := ndni.InterestTemplateFromPtr(unsafe.Pointer(&fp.tpl))
	tpl.Init(tplArgs...)

	if uintptr(fp.tpl.prefixL+1) >= unsafe.Sizeof(fp.tpl.prefixV) {
		return -1, errors.New("name too long")
	}
	fp.tpl.prefixV[fp.tpl.prefixL] = an.TtSegmentNameComponent
	// put SegmentNameComponent TLV-TYPE in the buffer so that it's checked in same memcmp

	rs := urcu.NewReadSide()
	defer rs.Close()
	fth := fetcher.fth[i%len(fetcher.fth)]
	C.cds_hlist_add_head_rcu(&fp.fthNode, &fth.c.head)
	fetcher.nActiveProcs++
	return i, nil
}

// Launch launches all fetch threads.
func (fetcher *Fetcher) Launch() {
	for _, fth := range fetcher.fth {
		fth.Launch()
	}
}

// Stop stops all fetch threads.
func (fetcher *Fetcher) Stop() {
	for _, fth := range fetcher.fth {
		fth.Stop()
	}
}

// Close deallocates data structures.
func (fetcher *Fetcher) Close() error {
	faceID := fetcher.Face().ID()
	for i, fp := range fetcher.fp {
		fetcher.RxQueue(i).Close()
		fetcher.Logic(i).Close()
		eal.Free(fp)
	}
	for _, fth := range fetcher.fth {
		eal.Free(fth.c)
	}
	delete(fetcherByFace, faceID)
	return nil
}

type fetchThread struct {
	ealthread.Thread
	c *C.FetchThread
}
