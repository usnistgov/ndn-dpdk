// Package tgconsumer implements a traffic generator consumer.
package tgconsumer

/*
#include "../../csrc/tgconsumer/rx.h"
#include "../../csrc/tgconsumer/tx.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go4.org/must"
)

// Consumer represents a traffic generator consumer instance.
type Consumer struct {
	Rx ealthread.Thread
	Tx ealthread.Thread

	rxC *C.TgConsumerRx
	txC *C.TgConsumerTx
}

// New creates a Consumer.
func New(face iface.Face, cfg Config) (*Consumer, error) {
	socket := face.NumaSocket()
	rxC := (*C.TgConsumerRx)(eal.Zmalloc("TgConsumerRx", C.sizeof_TgConsumerRx, socket))
	cfg.RxQueue.DisableCoDel = true
	if e := iface.PktQueueFromPtr(unsafe.Pointer(&rxC.rxQueue)).Init(cfg.RxQueue, socket); e != nil {
		eal.Free(rxC)
		return nil, nil
	}

	txC := (*C.TgConsumerTx)(eal.Zmalloc("TgConsumerTx", C.sizeof_TgConsumerTx, socket))
	txC.face = (C.FaceID)(face.ID())
	txC.interestMp = (*C.struct_rte_mempool)(ndni.InterestMempool.Get(socket).Ptr())
	C.pcg32_srandom_r(&txC.trafficRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	C.NonceGen_Init(&txC.nonceGen)

	var consumer Consumer
	consumer.rxC = rxC
	consumer.txC = txC
	consumer.Rx = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TgConsumerRx_Run), unsafe.Pointer(rxC)),
		ealthread.InitStopFlag(unsafe.Pointer(&rxC.stop)),
	)
	consumer.Tx = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.TgConsumerTx_Run), unsafe.Pointer(txC)),
		ealthread.InitStopFlag(unsafe.Pointer(&txC.stop)),
	)

	for i, pattern := range cfg.Patterns {
		if _, e := consumer.addPattern(pattern); e != nil {
			return nil, fmt.Errorf("pattern(%d): %s", i, e)
		}
	}
	consumer.SetInterval(cfg.Interval.Duration())
	return &consumer, nil
}

func (consumer *Consumer) addPattern(cfg Pattern) (index int, e error) {
	if consumer.rxC.nPatterns >= C.TGCONSUMER_MAX_PATTERNS {
		return -1, fmt.Errorf("cannot add more than %d patterns", C.TGCONSUMER_MAX_PATTERNS)
	}
	if cfg.Weight < 1 {
		cfg.Weight = 1
	}
	if consumer.txC.nWeights+C.uint16_t(cfg.Weight) >= C.TGCONSUMER_MAX_SUM_WEIGHT {
		return -1, fmt.Errorf("sum of weight cannot exceed %d", C.TGCONSUMER_MAX_SUM_WEIGHT)
	}
	index = int(consumer.rxC.nPatterns)
	if cfg.SeqNumOffset != 0 && index == 0 {
		return -1, errors.New("first pattern cannot have SeqNumOffset")
	}

	tplArgs := []interface{}{cfg.Prefix}
	if cfg.CanBePrefix {
		tplArgs = append(tplArgs, ndn.CanBePrefixFlag)
	}
	if cfg.MustBeFresh {
		tplArgs = append(tplArgs, ndn.MustBeFreshFlag)
	}
	if lifetime := cfg.InterestLifetime.Duration(); lifetime != 0 {
		tplArgs = append(tplArgs, lifetime)
	}
	if cfg.HopLimit != 0 {
		tplArgs = append(tplArgs, cfg.HopLimit)
	}

	consumer.clearCounter(index)
	rxP := &consumer.rxC.pattern[index]
	rxP.prefixLen = C.uint16_t(cfg.Prefix.Length())
	txP := &consumer.txC.pattern[index]
	ndni.InterestTemplateFromPtr(unsafe.Pointer(&txP.tpl)).Init(tplArgs...)

	txP.seqNum.compT = C.TtGenericNameComponent
	txP.seqNum.compL = C.uint8_t(C.sizeof_uint64_t)
	txP.seqNum.compV = C.uint64_t(rand.Uint64())
	txP.seqNumOffset = C.uint32_t(cfg.SeqNumOffset)

	consumer.rxC.nPatterns++
	for i := 0; i < cfg.Weight; i++ {
		consumer.txC.weight[consumer.txC.nWeights] = C.PingPatternId(index)
		consumer.txC.nWeights++
	}
	return index, nil
}

// Interval returns average Interest interval.
func (consumer *Consumer) Interval() time.Duration {
	return eal.FromTscDuration(int64(consumer.txC.burstInterval)) / C.TGCONSUMER_TX_BURST_SIZE
}

// SetInterval sets average Interest interval.
// TX thread transmits Interests in bursts, so the specified interval will be converted to
// a burst interval with equivalent traffic amount.
func (consumer *Consumer) SetInterval(interval time.Duration) {
	consumer.txC.burstInterval = C.TscDuration(eal.ToTscDuration(interval * C.TGCONSUMER_TX_BURST_SIZE))
}

// RxQueue returns the ingress queue.
func (consumer *Consumer) RxQueue() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&consumer.rxC.rxQueue))
}

// SetLCores assigns LCores to RX and TX threads.
func (consumer *Consumer) SetLCores(rxLCore, txLCore eal.LCore) {
	consumer.Rx.SetLCore(rxLCore)
	consumer.Tx.SetLCore(txLCore)
}

// Launch launches RX and TX threads.
func (consumer *Consumer) Launch() {
	consumer.rxC.runNum++
	consumer.txC.runNum = consumer.rxC.runNum
	consumer.Rx.Launch()
	consumer.Tx.Launch()
}

// Stop stops RX and TX threads.
func (consumer *Consumer) Stop(delay time.Duration) error {
	eTx := consumer.Tx.Stop()
	time.Sleep(delay)
	eRx := consumer.Rx.Stop()
	if eRx != nil || eTx != nil {
		return fmt.Errorf("RX %v; TX %v", eRx, eTx)
	}
	return nil
}

// Close closes the consumer.
// Both RX and TX threads must be stopped before calling this.
func (consumer *Consumer) Close() error {
	must.Close(consumer.RxQueue())
	eal.Free(consumer.rxC)
	eal.Free(consumer.txC)
	return nil
}
