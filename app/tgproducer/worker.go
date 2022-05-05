package tgproducer

/*
#include "../../csrc/tgproducer/producer.h"
*/
import "C"
import (
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/pcg32"
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

func (w *worker) setPatterns(patterns []Pattern, takeDataGenMbuf func() *pktmbuf.Packet) {
	w.freeDataGen()

	w.c.nPatterns = C.uint8_t(len(patterns))
	prefixes := ndni.NewLNamePrefixFilterBuilder(unsafe.Pointer(&w.c.prefixL), unsafe.Sizeof(w.c.prefixL),
		unsafe.Pointer(&w.c.prefixV), unsafe.Sizeof(w.c.prefixV))
	for i, pattern := range patterns {
		if e := prefixes.Append(pattern.Prefix); e != nil {
			panic(e)
		}
		pattern.assign(&w.c.pattern[i], takeDataGenMbuf)
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
		c: eal.Zmalloc[C.Tgp]("Tgp", C.sizeof_Tgp, socket),
	}

	if e := w.rxQueue().Init(rxqCfg, socket); e != nil {
		eal.Free(w.c)
		return nil, e
	}

	w.c.face = (C.FaceID)(faceID)
	pcg32.Init(unsafe.Pointer(&w.c.replyRng))
	(*ndni.Mempools)(unsafe.Pointer(&w.c.mp)).Assign(socket, ndni.DataMempool)

	w.ThreadWithCtrl = ealthread.NewThreadWithCtrl(
		cptr.Func0.C(C.Tgp_Run, w.c),
		unsafe.Pointer(&w.c.ctrl),
	)
	return w, nil
}

func (pattern Pattern) assign(c *C.TgpPattern, takeDataGenMbuf func() *pktmbuf.Packet) {
	*c = C.TgpPattern{
		nReplies: C.uint8_t(len(pattern.Replies)),
	}

	w := 0
	for k, reply := range pattern.Replies {
		reply.assign(&c.reply[k], takeDataGenMbuf)

		for j := 0; j < reply.Weight; j++ {
			c.weight[w] = C.TgpReplyID(k)
			w++
		}
	}
	c.nWeights = C.uint32_t(w)
}

func (reply Reply) assign(c *C.TgpReply, takeDataGenMbuf func() *pktmbuf.Packet) {
	kind := reply.Kind()
	*c = C.TgpReply{
		kind: C.uint8_t(kind),
	}
	switch kind {
	case ReplyNack:
		c.nackReason = C.uint8_t(reply.Nack)
	case ReplyData:
		dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&c.dataGen))
		reply.DataGenConfig.Apply(dataGen, takeDataGenMbuf())
	}
}
