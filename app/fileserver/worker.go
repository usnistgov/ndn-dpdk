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
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

const sizeofFileServerFd = C.sizeof_FileServerFd

type worker struct {
	ealthread.ThreadWithCtrl
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

func (w worker) counters() countersC {
	return *(*countersC)(unsafe.Pointer(&w.c.cnt))
}

func newWorker(faceID iface.ID, socket eal.NumaSocket, cfg Config) (w *worker, e error) {
	w = &worker{
		c: (*C.FileServer)(eal.Zmalloc("FileServer", C.sizeof_FileServer, socket)),
	}

	if e := w.rxQueue().Init(cfg.RxQueue, socket); e != nil {
		eal.Free(w.c)
		return nil, e
	}

	w.c.payloadMp = (*C.struct_rte_mempool)(ndni.PayloadMempool.Get(socket).Ptr())
	w.c.statValidity = (C.TscDuration)(cfg.tscStatValidity())
	w.c.face = (C.FaceID)(faceID)
	w.c.segmentLen = C.uint16_t(cfg.SegmentLen)
	w.c.payloadHeadroom = C.uint16_t(cfg.payloadHeadroom)
	w.c.uringCapacity = C.uint32_t(cfg.UringCapacity)
	w.c.uringCongMarkThreshold = C.uint32_t(cfg.UringCapacity / 2)
	w.c.uringWaitThreshold = C.uint32_t(cfg.UringCapacity / 4 * 3)
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
		cptr.Func0.C(unsafe.Pointer(C.FileServer_Run), w.c),
		unsafe.Pointer(&w.c.ctrl),
	)
	return w, nil
}
