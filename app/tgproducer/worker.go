package tgproducer

/*
#include "../../csrc/tgproducer/producer.h"
*/
import "C"
import (
	"math/rand"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type worker struct {
	ealthread.ThreadWithCtrl
	c *C.Tgp
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

func (w *worker) setPatterns(patterns []Pattern, dataGenVec *pktmbuf.Vector) {
	w.freeDataGen()

	w.c.nPatterns = C.uint8_t(len(patterns))
	for i, pattern := range patterns {
		c := &w.c.pattern[i]
		*c = C.TgpPattern{
			nReplies: C.uint8_t(len(pattern.Replies)),
		}
		prefixL := copy(cptr.AsByteSlice(&c.prefixBuffer), pattern.prefixV)
		c.prefix.value = &c.prefixBuffer[0]
		c.prefix.length = C.uint16_t(prefixL)

		w := 0
		for k, reply := range pattern.Replies {
			r := &c.reply[k]
			kind := reply.Kind()
			r.kind = C.uint8_t(kind)
			switch kind {
			case ReplyNack:
				r.nackReason = C.uint8_t(reply.Nack)
			case ReplyData:
				dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&r.dataGen))
				reply.DataGenConfig.Apply(dataGen, (*dataGenVec)[0])
				*dataGenVec = (*dataGenVec)[1:]
			}

			for j := 0; j < reply.Weight; j++ {
				c.weight[w] = C.TgpReplyID(k)
				w++
			}
		}
		c.nWeights = C.uint32_t(w)
	}
}

func (w *worker) close() error {
	w.freeDataGen()
	e := w.rxQueue().Close()
	eal.Free(w.c)
	return e
}

func (w *worker) freeDataGen() {
	for _, pattern := range w.c.pattern[:w.c.nPatterns] {
		for _, r := range pattern.reply[:pattern.nReplies] {
			dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&r.dataGen))
			dataGen.Close()
		}
	}
}

func newWorker(faceID iface.ID, socket eal.NumaSocket, rxqCfg iface.PktQueueConfig) (w *worker, e error) {
	w = &worker{
		c: (*C.Tgp)(eal.Zmalloc("Tgp", C.sizeof_Tgp, socket)),
	}

	rxQueue := iface.PktQueueFromPtr(unsafe.Pointer(&w.c.rxQueue))
	if e := rxQueue.Init(rxqCfg, socket); e != nil {
		eal.Free(w.c)
		return nil, e
	}

	w.c.face = (C.FaceID)(faceID)
	C.pcg32_srandom_r(&w.c.replyRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	(*ndni.Mempools)(unsafe.Pointer(&w.c.mp)).Assign(socket, ndni.DataMempool)

	w.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(unsafe.Pointer(C.Tgp_Run), w.c),
		unsafe.Pointer(&w.c.ctrl),
	)
	return w, nil
}
