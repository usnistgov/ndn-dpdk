package fileserver

/*
#include "../../csrc/fileserver/server.h"
#include "../../csrc/fileserver/fd.h"
*/
import "C"
import (
	"unsafe"

	binutils "github.com/jfoster/binary-utilities"
	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/mempool"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
)

type worker struct {
	ealthread.ThreadWithCtrl
	c    *C.FileServer
	fdMp *mempool.Mempool
}

var (
	_ ealthread.ThreadWithRole     = (*worker)(nil)
	_ ealthread.ThreadWithLoadStat = (*worker)(nil)
)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (worker) ThreadRole() string {
	return tgdef.RoleProducer
}

// NumaSocket implements eal.WithNumaSocket interface.
func (w worker) NumaSocket() eal.NumaSocket {
	return w.face().NumaSocket()
}

func (w worker) face() iface.Face {
	return iface.Get(iface.ID(w.c.face))
}

func (w worker) rxQueue() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&w.c.rxQueue))
}

func (w *worker) close() error {
	errs := []error{}
	if w.fdMp != nil {
		errs = append(errs, w.fdMp.Close())
		w.fdMp = nil
	}
	errs = append(errs, w.rxQueue().Close())
	eal.Free(w.c)
	w.c = nil
	return multierr.Combine(errs...)
}

func (w worker) addToCounters(cnt *Counters) {
	cnt.ReqRead += uint64(w.c.cnt.reqRead)
	cnt.ReqLs += uint64(w.c.cnt.reqLs)
	cnt.ReqMetadata += uint64(w.c.cnt.reqMetadata)
	cnt.FdNew += uint64(w.c.cnt.fdNew)
	cnt.FdNotFound += uint64(w.c.cnt.fdNotFound)
	cnt.FdUpdateStat += uint64(w.c.cnt.fdUpdateStat)
	cnt.FdClose += uint64(w.c.cnt.fdClose)
	cnt.UringAllocError += uint64(w.c.ur.nAllocErrs)
	cnt.UringSubmitted += uint64(w.c.ur.nSubmitted)
	cnt.UringSubmitNonBlock += uint64(w.c.ur.nSubmitNonBlock)
	cnt.UringSubmitWait += uint64(w.c.ur.nSubmitWait)
	cnt.UringCqeFail += uint64(w.c.cnt.cqeFail)
}

func newWorker(faceID iface.ID, socket eal.NumaSocket, cfg Config) (w *worker, e error) {
	w = &worker{
		c: eal.Zmalloc[C.FileServer]("FileServer", C.sizeof_FileServer, socket),
	}

	if e := w.rxQueue().Init(cfg.RxQueue, socket); e != nil {
		w.close()
		return nil, e
	}

	if w.fdMp, e = mempool.New(mempool.Config{
		Capacity:       cfg.OpenFds,
		ElementSize:    C.sizeof_FileServerFd,
		Socket:         socket,
		SingleProducer: true,
		SingleConsumer: true,
	}); e != nil {
		w.close()
		return nil, e
	}

	w.c.payloadMp = (*C.struct_rte_mempool)(ndni.PayloadMempool.Get(socket).Ptr())
	w.c.fdMp = (*C.struct_rte_mempool)(w.fdMp.Ptr())
	w.c.statValidity = (C.TscDuration)(cfg.tscStatValidity())
	w.c.face = (C.FaceID)(faceID)
	w.c.segmentLen = C.uint16_t(cfg.SegmentLen)
	w.c.payloadHeadroom = C.uint16_t(cfg.payloadHeadroom)
	w.c.uringCapacity = C.uint32_t(cfg.UringCapacity)
	w.c.uringCongestionLbound = C.uint32_t(cfg.uringCongestionLbound)
	w.c.uringWaitLbound = C.uint32_t(cfg.uringWaitLbound)
	w.c.nFdHtBuckets = C.uint32_t(binutils.PrevPowerOfTwo(int64(cfg.OpenFds)))
	w.c.fdQCapacity = C.uint16_t(cfg.KeepFds)

	prefixes := ndni.NewLNamePrefixFilterBuilder(unsafe.Pointer(&w.c.mountPrefixL), unsafe.Sizeof(w.c.mountPrefixL),
		unsafe.Pointer(&w.c.mountPrefixV), unsafe.Sizeof(w.c.mountPrefixV))
	for i, m := range cfg.Mounts {
		w.c.dfd[i] = C.int(*m.dfd)
		w.c.mountPrefixComps[i] = C.int16_t(len(m.Prefix))
		prefixes.Append(m.Prefix)
	}

	w.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.FileServer_Run, w.c),
		unsafe.Pointer(&w.c.ctrl),
	)
	return w, nil
}

// Counters contains file server counters.
type Counters struct {
	ReqRead             uint64 `json:"reqRead" gqldesc:"Received read requests."`
	ReqLs               uint64 `json:"reqLs" gqldesc:"Received directory listing requests."`
	ReqMetadata         uint64 `json:"reqMetadata" gqldesc:"Received metadata requests."`
	FdNew               uint64 `json:"fdNew" gqldesc:"Successfully opened file descriptors."`
	FdNotFound          uint64 `json:"fdNotFound" gqldesc:"File not found."`
	FdUpdateStat        uint64 `json:"fdUpdateStat" gqldesc:"Update stat on already open file descriptors."`
	FdClose             uint64 `json:"fdClose" gqldesc:"Closed file descriptors."`
	UringAllocError     uint64 `json:"uringAllocErrs" gqldesc:"uring SQE allocation errors."`
	UringSubmitted      uint64 `json:"uringSubmitted" gqldesc:"uring submitted SQEs."`
	UringSubmitNonBlock uint64 `json:"uringSubmitNonBlock" gqldesc:"uring non-blocking submission batches."`
	UringSubmitWait     uint64 `json:"uringSubmitWait" gqldesc:"uring waiting submission batches."`
	UringCqeFail        uint64 `json:"cqeFail" gqldesc:"uring failed CQEs."`
}
