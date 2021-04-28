// Package tgproducer implements a traffic generator producer.
package tgproducer

/*
#include "../../csrc/tgproducer/producer.h"
*/
import "C"
import (
	"math/rand"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

// Producer represents a traffic generator producer thread.
type Producer struct {
	ealthread.Thread
	c        *C.Tgp
	patterns []Pattern
}

// Patterns returns traffic patterns.
func (p Producer) Patterns() []Pattern {
	return p.patterns
}

// SetPatterns sets new traffic patterns.
// This can only be used when the thread is stopped.
func (p *Producer) SetPatterns(inputPatterns []Pattern) error {
	if len(inputPatterns) == 0 {
		return ErrNoPattern
	}
	if len(inputPatterns) > MaxPatterns {
		return ErrTooManyPatterns
	}
	patterns := []Pattern{}
	nDataGen := 0
	for _, pattern := range inputPatterns {
		sumWeight, nData := pattern.applyDefaults()
		if sumWeight > MaxSumWeight {
			return ErrTooManyWeights
		}
		nDataGen += nData
		if len(pattern.prefixV) > ndni.NameMaxLength {
			return ErrPrefixTooLong
		}
		patterns = append(patterns, pattern)
	}

	payloadMp := ndni.PayloadMempool.Get(p.Face().NumaSocket())
	dataGenVec, e := payloadMp.Alloc(nDataGen)
	if e != nil {
		return e
	}

	if p.IsRunning() {
		return ealthread.ErrRunning
	}

	p.patterns = patterns
	p.c.nPatterns = C.uint8_t(len(patterns))
	for i, pattern := range patterns {
		c := &p.c.pattern[i]
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
				data := ndn.MakeData(reply.Suffix, reply.FreshnessPeriod.Duration(), make([]byte, reply.PayloadLen))
				dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&r.dataGen))
				dataGen.Init(dataGenVec[0], data)
				dataGenVec = dataGenVec[1:]
			}

			for j := 0; j < reply.Weight; j++ {
				c.weight[w] = C.TgpReplyID(k)
				w++
			}
		}
		c.nWeights = C.uint32_t(w)
	}
	return nil
}

// RxQueue returns the ingress queue.
func (p Producer) RxQueue() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&p.c.rxQueue))
}

// Face returns the associated face.
func (p Producer) Face() iface.Face {
	return iface.Get(iface.ID(p.c.face))
}

// Close closes the producer.
// The thread must be stopped before calling this.
func (p *Producer) Close() error {
	p.Stop()
	must.Close(p.RxQueue())
	eal.Free(p.c)
	return nil
}

// New creates a Producer.
func New(face iface.Face, rxqCfg iface.PktQueueConfig) (p *Producer, e error) {
	faceID := face.ID()
	socket := face.NumaSocket()
	p = &Producer{
		c: (*C.Tgp)(eal.Zmalloc("Tgp", C.sizeof_Tgp, socket)),
	}

	rxqCfg.DisableCoDel = true
	rxQueue := iface.PktQueueFromPtr(unsafe.Pointer(&p.c.rxQueue))
	if e := rxQueue.Init(rxqCfg, socket); e != nil {
		eal.Free(p.c)
		return nil, nil
	}

	p.c.face = (C.FaceID)(faceID)
	C.pcg32_srandom_r(&p.c.replyRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))

	(*ndni.Mempools)(unsafe.Pointer(&p.c.mp)).Assign(socket, ndni.DataMempool)

	p.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.Tgp_Run), unsafe.Pointer(p.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&p.c.stop)),
	)

	return p, nil
}
