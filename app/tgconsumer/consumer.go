// Package tgconsumer implements a traffic generator consumer.
package tgconsumer

/*
#include "../../csrc/tgconsumer/rx.h"
#include "../../csrc/tgconsumer/tx.h"
*/
import "C"
import (
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
	"go4.org/must"
)

// RoleConsumer indicates consumer thread role.
const RoleConsumer = "CONSUMER"

type worker struct {
	ealthread.Thread
}

// ThreadRole implements ealthread.ThreadWithRole interface.
func (worker) ThreadRole() string {
	return RoleConsumer
}

// Consumer represents a traffic generator consumer instance.
type Consumer struct {
	rx       worker
	tx       worker
	rxC      *C.TgcRx
	txC      *C.TgcTx
	patterns []Pattern
}

// Workers returns worker threads.
func (c Consumer) Workers() []ealthread.ThreadWithRole {
	return []ealthread.ThreadWithRole{c.rx, c.tx}
}

// Patterns returns traffic patterns.
func (c Consumer) Patterns() []Pattern {
	return c.patterns
}

// SetPatterns sets new traffic patterns.
// This can only be used when both RX and TX threads are stopped.
func (c *Consumer) SetPatterns(inputPatterns []Pattern) error {
	if len(inputPatterns) == 0 {
		return ErrNoPattern
	}
	if len(inputPatterns) > MaxPatterns {
		return ErrTooManyPatterns
	}
	patterns := []Pattern{}
	nWeights := 0
	for i, pattern := range inputPatterns {
		pattern.applyDefaults()
		patterns = append(patterns, pattern)
		if pattern.SeqNumOffset != 0 && i == 0 {
			return ErrFirstSeqNumOffset
		}
		nWeights += pattern.Weight
	}
	if nWeights > MaxSumWeight {
		return ErrTooManyWeights
	}

	if c.rx.IsRunning() || c.tx.IsRunning() {
		return ealthread.ErrRunning
	}

	c.patterns = patterns
	c.rxC.nPatterns = C.uint8_t(len(patterns))
	c.txC.nWeights = C.uint32_t(nWeights)
	w := 0
	for i, pattern := range patterns {
		rxP := &c.rxC.pattern[i]
		*rxP = C.TgcRxPattern{
			prefixLen: C.uint16_t(pattern.Prefix.Length()),
		}

		txP := &c.txC.pattern[i]
		*txP = C.TgcTxPattern{
			seqNum: C.TgcSeqNum{
				compT: C.TtGenericNameComponent,
				compL: C.uint8_t(C.sizeof_uint64_t),
				compV: C.uint64_t(rand.Uint64()),
			},
			seqNumOffset: C.uint32_t(pattern.SeqNumOffset),
		}
		pattern.initInterestTemplate(ndni.InterestTemplateFromPtr(unsafe.Pointer(&txP.tpl)))

		for j := 0; j < pattern.Weight; j++ {
			c.txC.weight[w] = C.TgcPatternID(i)
			w++
		}

		c.clearCounter(i)
	}
	return nil
}

// Interval returns average Interest interval.
func (c Consumer) Interval() time.Duration {
	return eal.FromTscDuration(int64(c.txC.burstInterval)) / iface.MaxBurstSize
}

// SetInterval sets average Interest interval.
// TX thread transmits Interests in bursts, so the specified interval will be converted to
// a burst interval with equivalent traffic amount.
// This can only be used when both RX and TX threads are stopped.
func (c *Consumer) SetInterval(interval time.Duration) error {
	if c.rx.IsRunning() || c.tx.IsRunning() {
		return ealthread.ErrRunning
	}

	c.txC.burstInterval = C.TscDuration(eal.ToTscDuration(interval * iface.MaxBurstSize))
	return nil
}

// RxQueue returns the ingress queue.
func (c Consumer) RxQueue() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&c.rxC.rxQueue))
}

// Face returns the associated face.
func (c Consumer) Face() iface.Face {
	return iface.Get(iface.ID(c.txC.face))
}

// AllocLCores allocates worker lcores.
// This can only be used when all workers are stopped.
func (p *Consumer) AllocLCores(allocator *ealthread.Allocator) error {
	return multierr.Append(
		allocator.AllocThread(p.rx),
		allocator.AllocThread(p.tx),
	)
}

// Launch launches RX and TX threads.
func (c *Consumer) Launch() {
	c.rxC.runNum++
	c.txC.runNum = c.rxC.runNum
	c.rx.Launch()
	c.tx.Launch()
}

// Stop stops RX and TX threads.
func (consumer *Consumer) Stop(delay time.Duration) error {
	eTx := consumer.tx.Stop()
	time.Sleep(delay)
	eRx := consumer.rx.Stop()
	return multierr.Append(eTx, eRx)
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	c.Stop(0)
	must.Close(c.RxQueue())
	eal.Free(c.rxC)
	eal.Free(c.txC)
	return nil
}

// New creates a Consumer.
func New(face iface.Face, rxqCfg iface.PktQueueConfig) (c *Consumer, e error) {
	socket := face.NumaSocket()
	c = &Consumer{
		rxC: (*C.TgcRx)(eal.Zmalloc("TgcRx", C.sizeof_TgcRx, socket)),
		txC: (*C.TgcTx)(eal.Zmalloc("TgcTx", C.sizeof_TgcTx, socket)),
	}

	rxqCfg.DisableCoDel = true
	if e = c.RxQueue().Init(rxqCfg, socket); e != nil {
		must.Close(c)
		return nil, fmt.Errorf("error initializing RxQueue %w", e)
	}

	c.txC.face = (C.FaceID)(face.ID())
	c.txC.interestMp = (*C.struct_rte_mempool)(ndni.InterestMempool.Get(socket).Ptr())
	C.pcg32_srandom_r(&c.txC.trafficRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	C.NonceGen_Init(&c.txC.nonceGen)

	c.rx.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TgcRx_Run), unsafe.Pointer(c.rxC)),
		ealthread.InitStopFlag(unsafe.Pointer(&c.rxC.stop)),
	)
	c.tx.Thread = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TgcTx_Run), unsafe.Pointer(c.txC)),
		ealthread.InitStopFlag(unsafe.Pointer(&c.txC.stop)),
	)

	c.SetInterval(time.Millisecond)
	return c, nil
}
