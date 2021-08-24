package fileserver

/*
#include "../../csrc/fileserver/server.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type worker struct {
	ealthread.Thread
	c *C.FileServer
}

var (
	_ ealthread.ThreadWithRole     = (*worker)(nil)
	_ ealthread.ThreadWithLoadStat = (*worker)(nil)
)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (worker) ThreadRole() string {
	return tgdef.RoleProducer
}

// ThreadLoadStat implements ealthread.ThreadWithLoadStat interface.
func (w worker) ThreadLoadStat() ealthread.LoadStat {
	return ealthread.LoadStatFromPtr(unsafe.Pointer(&w.c.loadStat))
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
	e := w.rxQueue().Close()
	eal.Free(w.c)
	return e
}

func newWorker(faceID iface.ID, socket eal.NumaSocket, cfg Config) (w *worker, e error) {
	w = &worker{
		c: (*C.FileServer)(eal.Zmalloc("FileServer", C.sizeof_FileServer, socket)),
	}

	rxQueue := iface.PktQueueFromPtr(unsafe.Pointer(&w.c.rxQueue))
	if e := rxQueue.Init(cfg.RxQueue, socket); e != nil {
		eal.Free(w.c)
		return nil, e
	}

	w.c.payloadMp = (*C.struct_rte_mempool)(ndni.PayloadMempool.Get(socket).Ptr())
	w.c.face = (C.FaceID)(faceID)
	w.c.segmentLen = C.uint16_t(cfg.SegmentLen)
	w.c.payloadHeadroom = C.uint16_t(cfg.payloadHeadroom)
	w.c.uringCapacity = C.uint32_t(cfg.UringCapacity)

	prefixes := ndni.NewLNamePrefixFilterBuilder(unsafe.Pointer(&w.c.prefixL), unsafe.Sizeof(w.c.prefixL),
		unsafe.Pointer(&w.c.prefixV), unsafe.Sizeof(w.c.prefixV))
	for i, m := range cfg.Mounts {
		w.c.dfd[i] = C.int(*m.dfd)
		prefixes.Append(m.Prefix)
	}

	w.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.FileServer_Run), unsafe.Pointer(w.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&w.c.stop)),
	)
	return w, nil
}
