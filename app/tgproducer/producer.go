// Package tgproducer implements a traffic generator producer.
package tgproducer

/*
#include "../../csrc/tgproducer/producer.h"
*/
import "C"
import (
	"fmt"
	"math/rand"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

// Producer represents a traffic generator producer thread.
type Producer struct {
	ealthread.Thread
	c         *C.TgProducer
	payloadMp *pktmbuf.Pool
}

// New creates a Producer.
func New(face iface.Face, index int, cfg Config) (producer *Producer, e error) {
	faceID := face.ID()
	socket := face.NumaSocket()
	producer = &Producer{
		c:         (*C.TgProducer)(eal.Zmalloc("TgProducer", C.sizeof_TgProducer, socket)),
		payloadMp: ndni.PayloadMempool.Get(socket),
	}

	cfg.RxQueue.DisableCoDel = true
	if e := iface.PktQueueFromPtr(unsafe.Pointer(&producer.c.rxQueue)).Init(cfg.RxQueue, socket); e != nil {
		eal.Free(producer.c)
		return nil, nil
	}

	producer.c.face = (C.FaceID)(faceID)
	producer.c.wantNackNoRoute = C.bool(cfg.Nack)
	C.pcg32_srandom_r(&producer.c.replyRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))

	(*ndni.Mempools)(unsafe.Pointer(&producer.c.mp)).Assign(socket, ndni.DataMempool)

	producer.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TgProducer_Run), unsafe.Pointer(producer.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&producer.c.stop)),
	)

	for i, pattern := range cfg.Patterns {
		if _, e := producer.addPattern(pattern); e != nil {
			return nil, fmt.Errorf("pattern(%d): %s", i, e)
		}
	}

	return producer, nil
}

func (producer *Producer) addPattern(cfg Pattern) (index int, e error) {
	if producer.c.nPatterns >= C.TGPRODUCER_MAX_PATTERNS {
		return -1, fmt.Errorf("cannot add more than %d patterns", C.TGPRODUCER_MAX_PATTERNS)
	}
	if len(cfg.Replies) < 1 || len(cfg.Replies) > C.TGPRODUCER_MAX_REPLIES {
		return -1, fmt.Errorf("must have between 1 and %d reply definitions", C.TGPRODUCER_MAX_REPLIES)
	}

	index = int(producer.c.nPatterns)
	patternC := &producer.c.pattern[index]
	*patternC = C.TgProducerPattern{}

	prefixV, _ := cfg.Prefix.MarshalBinary()
	if len(prefixV) > len(patternC.prefixBuffer) {
		return -1, fmt.Errorf("prefix too long")
	}
	for i, b := range prefixV {
		patternC.prefixBuffer[i] = C.uint8_t(b)
	}
	patternC.prefix.value = &patternC.prefixBuffer[0]
	patternC.prefix.length = C.uint16_t(len(prefixV))

	for i, reply := range cfg.Replies {
		if reply.Weight < 1 {
			reply.Weight = 1
		}
		if patternC.nWeights+C.uint16_t(reply.Weight) >= C.TGPRODUCER_MAX_SUM_WEIGHT {
			return -1, fmt.Errorf("sum of weight cannot exceed %d", C.TGPRODUCER_MAX_SUM_WEIGHT)
		}
		for j := 0; j < reply.Weight; j++ {
			patternC.weight[patternC.nWeights] = C.PingReplyId(i)
			patternC.nWeights++
		}

		replyC := &patternC.reply[i]
		switch {
		case reply.Timeout:
			replyC.kind = C.TGPRODUCER_REPLY_TIMEOUT
		case reply.Nack != an.NackNone:
			replyC.kind = C.TGPRODUCER_REPLY_NACK
			replyC.nackReason = C.uint8_t(reply.Nack)
		default:
			replyC.kind = C.TGPRODUCER_REPLY_DATA
			vec, e := producer.payloadMp.Alloc(1)
			if e != nil {
				return -1, fmt.Errorf("cannot allocate from payloadMp for reply definition %d", i)
			}
			data := ndn.MakeData(reply.Suffix, reply.FreshnessPeriod.Duration(), make([]byte, reply.PayloadLen))
			dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&replyC.dataGen))
			dataGen.Init(vec[0], data)
		}
	}
	patternC.nReplies = C.uint16_t(len(cfg.Replies))

	producer.c.nPatterns++
	return index, nil
}

// RxQueue returns the ingress queue.
func (producer *Producer) RxQueue() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&producer.c.rxQueue))
}

// Close closes the producer.
// The thread must be stopped before calling this.
func (producer *Producer) Close() error {
	producer.Stop()
	must.Close(producer.RxQueue())
	eal.Free(producer.c)
	return nil
}
