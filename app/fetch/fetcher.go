// Package fetch simulates bulk file transfer traffic patterns.
package fetch

/*
#include "../../csrc/fetch/fetcher.h"
*/
import "C"
import (
	"errors"
	"math"
	"unsafe"

	mathpkg "github.com/pkg/math"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/urcu"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
)

// FetcherConfig contains Fetcher configuration.
type FetcherConfig struct {
	NThreads       int                  `json:"nThreads,omitempty"`
	NProcs         int                  `json:"nProcs,omitempty"`
	RxQueue        iface.PktQueueConfig `json:"rxQueue,omitempty"`
	WindowCapacity int                  `json:"windowCapacity,omitempty"`
}

func (cfg *FetcherConfig) applyDefaults() {
	cfg.NThreads = mathpkg.MaxInt(1, cfg.NThreads)
	cfg.NProcs = mathpkg.MaxInt(1, cfg.NProcs)
	cfg.RxQueue.DisableCoDel = true
}

// RoleConsumer indicates consumer thread role.
const RoleConsumer = "CONSUMER"

type worker struct {
	ealthread.Thread
	c *C.FetchThread
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (worker) ThreadRole() string {
	return RoleConsumer
}

// NumaSocket implements eal.WithNumaSocket interface.
func (w worker) NumaSocket() eal.NumaSocket {
	return w.face().NumaSocket()
}

func (w worker) face() iface.Face {
	return iface.Get(iface.ID(w.c.face))
}

// Fetcher controls fetch threads and fetch procedures on a face.
// A fetch procedure retrieves Data under a single name prefix, and has independent congestion control.
// A fetch thread runs on an lcore, and can serve multiple fetch procedures.
type Fetcher struct {
	workers      []*worker
	fp           []*C.FetchProc
	nActiveProcs int
}

// New creates a Fetcher.
func New(face iface.Face, cfg FetcherConfig) (*Fetcher, error) {
	cfg.applyDefaults()
	if cfg.NProcs >= math.MaxUint8 { // InputDemux dispatches on 1-octet of PIT token
		return nil, errors.New("too many procs")
	}

	faceID := face.ID()
	socket := face.NumaSocket()
	interestMp := (*C.struct_rte_mempool)(ndni.InterestMempool.Get(socket).Ptr())

	fetcher := &Fetcher{
		workers: make([]*worker, cfg.NThreads),
		fp:      make([]*C.FetchProc, cfg.NProcs),
	}
	for i := range fetcher.workers {
		w := &worker{
			c: (*C.FetchThread)(eal.Zmalloc("FetchThread", C.sizeof_FetchThread, socket)),
		}
		w.c.face = (C.FaceID)(faceID)
		w.c.interestMp = interestMp
		C.NonceGen_Init(&w.c.nonceGen)
		w.Thread = ealthread.New(
			cptr.Func0.C(unsafe.Pointer(C.FetchThread_Run), unsafe.Pointer(w.c)),
			ealthread.InitStopFlag(unsafe.Pointer(&w.c.stop)),
		)
		fetcher.workers[i] = w
	}

	for i := range fetcher.fp {
		fp := (*C.FetchProc)(eal.Zmalloc("FetchProc", C.sizeof_FetchProc, socket))
		if e := iface.PktQueueFromPtr(unsafe.Pointer(&fp.rxQueue)).Init(cfg.RxQueue, socket); e != nil {
			return nil, e
		}
		fp.pitToken = C.uint8_t(i)
		fetcher.fp[i] = fp
		fetcher.Logic(i).Init(cfg.WindowCapacity, socket)
	}

	return fetcher, nil
}

// Face returns the face.
func (fetcher *Fetcher) Face() iface.Face {
	return fetcher.workers[0].face()
}

// Workers returns worker threads.
func (fetcher Fetcher) Workers() (list []ealthread.ThreadWithRole) {
	for _, w := range fetcher.workers {
		list = append(list, w)
	}
	return list
}

// ConnectRxQueues connects Data+Nack InputDemux to RxQueues.
func (fetcher *Fetcher) ConnectRxQueues(demuxD, demuxN *iface.InputDemux) {
	demuxD.InitToken(0)
	demuxN.InitToken(0)
	for i := range fetcher.fp {
		q := fetcher.rxQueue(i)
		demuxD.SetDest(i, q)
		demuxN.SetDest(i, q)
	}
}

func (fetcher *Fetcher) rxQueue(i int) *iface.PktQueue {
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
	for _, fth := range fetcher.workers {
		fth.c.head.next = nil
	}
	for i := range fetcher.fp {
		fetcher.Logic(i).Reset()
	}
	fetcher.nActiveProcs = 0
}

// AddTemplate sets name prefix and other InterestTemplate arguments.
// Return index of fetch procedure.
func (fetcher *Fetcher) AddTemplate(tplCfg ndni.InterestTemplateConfig) (i int, e error) {
	i = fetcher.nActiveProcs
	if i >= len(fetcher.fp) {
		return -1, errors.New("too many templates")
	}

	fp := fetcher.fp[i]
	tpl := ndni.InterestTemplateFromPtr(unsafe.Pointer(&fp.tpl))
	tplCfg.Apply(tpl)

	if uintptr(fp.tpl.prefixL+1) >= unsafe.Sizeof(fp.tpl.prefixV) {
		return -1, errors.New("name too long")
	}
	fp.tpl.prefixV[fp.tpl.prefixL] = an.TtSegmentNameComponent
	// put SegmentNameComponent TLV-TYPE in the buffer so that it's checked in same memcmp

	rs := urcu.NewReadSide()
	defer rs.Close()
	fth := fetcher.workers[i%len(fetcher.workers)]
	C.cds_hlist_add_head_rcu(&fp.fthNode, &fth.c.head)
	fetcher.nActiveProcs++
	return i, nil
}

// Launch launches all fetch threads.
func (fetcher *Fetcher) Launch() {
	for _, fth := range fetcher.workers {
		fth.Launch()
	}
}

// Stop stops all fetch threads.
func (fetcher *Fetcher) Stop() error {
	errs := []error{}
	for _, fth := range fetcher.workers {
		errs = append(errs, fth.Stop())
	}
	return multierr.Combine(errs...)
}

// Close deallocates data structures.
func (fetcher *Fetcher) Close() error {
	errs := []error{
		fetcher.Stop(),
	}
	for i, fp := range fetcher.fp {
		errs = append(errs,
			fetcher.rxQueue(i).Close(),
			fetcher.Logic(i).Close(),
		)
		eal.Free(fp)
	}
	for _, fth := range fetcher.workers {
		eal.Free(fth.c)
	}
	return multierr.Combine(errs...)
}
