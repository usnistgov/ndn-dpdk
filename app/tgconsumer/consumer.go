// Package tgconsumer implements a traffic generator consumer.
package tgconsumer

/*
#include "../../csrc/tgconsumer/rx.h"
#include "../../csrc/tgconsumer/tx.h"

TgcTxDigestPattern** c_TgcTxPattern_digest(TgcTxPattern* pattern) { return &pattern->digest; }
uint32_t* c_TgcTxPattern_seqNumOffset(TgcTxPattern* pattern) { return &pattern->seqNumOffset; }
void c_TgcTxDigestPattern_putPrefix(TgcTxDigestPattern* dp, uint16_t length, const uint8_t* value)
{
	dp->prefix.length = length;
	dp->prefix.value = rte_memcpy(RTE_PTR_ADD(dp, sizeof(*dp)), value, length);
}
*/
import "C"
import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/cryptodev"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndn/tlv"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/multierr"
	"go4.org/must"
)

// RoleConsumer indicates consumer thread role.
const RoleConsumer = "CONSUMER"

type worker struct {
	ealthread.Thread
	loadStat *C.ThreadLoadStat
}

var (
	_ ealthread.ThreadWithRole     = (*worker)(nil)
	_ ealthread.ThreadWithLoadStat = (*worker)(nil)
)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (worker) ThreadRole() string {
	return RoleConsumer
}

// ThreadLoadStat implements ealthread.ThreadWithLoadStat interface.
func (w worker) ThreadLoadStat() ealthread.LoadStat {
	return ealthread.LoadStatFromPtr(unsafe.Pointer(w.loadStat))
}

// Consumer represents a traffic generator consumer instance.
type Consumer struct {
	rx       worker
	tx       worker
	rxC      *C.TgcRx
	txC      *C.TgcTx
	patterns []Pattern

	digestOpPool *cryptodev.OpPool
	digestCrypto *cryptodev.CryptoDev
	dPatterns    []*C.TgcTxDigestPattern
}

func (c Consumer) socket() eal.NumaSocket {
	return c.Face().NumaSocket()
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
func (c *Consumer) SetPatterns(inputPatterns []Pattern) (e error) {
	if len(inputPatterns) == 0 {
		return ErrNoPattern
	}
	if len(inputPatterns) > MaxPatterns {
		return ErrTooManyPatterns
	}
	patterns := []Pattern{}
	nWeights, nDigestPatterns := 0, 0
	for i, pattern := range inputPatterns {
		pattern.applyDefaults()
		patterns = append(patterns, pattern)
		if pattern.SeqNumOffset != 0 && i == 0 {
			return ErrFirstSeqNumOffset
		}
		nWeights += pattern.Weight
		if pattern.Digest != nil {
			nDigestPatterns++
		}
	}
	if nWeights > MaxSumWeight {
		return ErrTooManyWeights
	}

	if c.rx.IsRunning() || c.tx.IsRunning() {
		return ealthread.ErrRunning
	}

	if e := c.prepareDigest(nDigestPatterns); e != nil {
		return fmt.Errorf("prepareDigest %w", e)
	}
	var dataGenVec pktmbuf.Vector
	if nDigestPatterns > 0 {
		payloadMp := ndni.PayloadMempool.Get(c.socket())
		dataGenVec, e = payloadMp.Alloc(nDigestPatterns)
		if e != nil {
			return e
		}
	}

	c.patterns = patterns
	c.rxC.nPatterns = C.uint8_t(len(patterns))
	c.txC.nWeights = C.uint32_t(nWeights)
	w := 0
	for i, pattern := range patterns {
		c.assignPattern(i, pattern, dataGenVec)

		for j := 0; j < pattern.Weight; j++ {
			c.txC.weight[w] = C.uint8_t(i)
			w++
		}

		c.clearCounter(i)
	}
	return nil
}

func (c *Consumer) assignPattern(i int, pattern Pattern, dataGenVec pktmbuf.Vector) {
	rxP := &c.rxC.pattern[i]
	*rxP = C.TgcRxPattern{
		prefixLen: C.uint16_t(pattern.Prefix.Length()),
	}

	txP := &c.txC.pattern[i]
	*txP = C.TgcTxPattern{
		seqNumT: an.TtGenericNameComponent,
		seqNumL: C.uint8_t(C.sizeof_uint64_t),
		seqNumV: C.uint64_t(rand.Uint64()),
		digestT: an.TtImplicitSha256DigestComponent,
		digestL: ndni.ImplicitDigestLength,
	}
	pattern.InterestTemplateConfig.Apply(ndni.InterestTemplateFromPtr(unsafe.Pointer(&txP.tpl)))

	switch {
	case pattern.Digest != nil:
		c.assignDigestPattern(pattern, txP, dataGenVec)
	case pattern.SeqNumOffset != 0:
		txP.makeSuffix = C.TgcTxPattern_MakeSuffix(C.TgcTxPattern_MakeSuffix_Offset)
		*C.c_TgcTxPattern_seqNumOffset(txP) = C.uint32_t(pattern.SeqNumOffset)
	default:
		txP.makeSuffix = C.TgcTxPattern_MakeSuffix(C.TgcTxPattern_MakeSuffix_Increment)
	}
}

func (c *Consumer) assignDigestPattern(pattern Pattern, txP *C.TgcTxPattern, dataGenVec pktmbuf.Vector) {
	txP.makeSuffix = C.TgcTxPattern_MakeSuffix(C.TgcTxPattern_MakeSuffix_Digest)

	name := append(ndn.Name{}, pattern.Prefix...)
	seqNumV := make([]byte, 8)
	binary.LittleEndian.PutUint64(seqNumV, uint64(txP.seqNumV))
	name = append(name, ndn.MakeNameComponent(an.TtGenericNameComponent, seqNumV))
	nameV, _ := tlv.EncodeValueOnly(name.Field())

	d := len(c.dPatterns)
	dp := (*C.TgcTxDigestPattern)(eal.Zmalloc("TgcTxDigestPattern", C.sizeof_TgcTxDigestPattern+len(nameV), c.socket()))
	(*ndni.Mempools)(unsafe.Pointer(&dp.dataMp)).Assign(c.socket(), ndni.DataMempool)
	dp.opPool = (*C.struct_rte_mempool)(c.digestOpPool.Ptr())
	c.digestCrypto.QueuePairs()[d].CopyToC(unsafe.Pointer(&dp.cqp))

	dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&dp.dataGen))
	pattern.Digest.Apply(dataGen, dataGenVec[d])

	nameVC := C.CBytes(nameV)
	defer C.free(nameVC)
	C.c_TgcTxDigestPattern_putPrefix(dp, C.uint16_t(len(nameV)), (*C.uint8_t)(nameVC))

	*C.c_TgcTxPattern_digest(txP) = dp
	c.dPatterns = append(c.dPatterns, dp)
}

func (c *Consumer) prepareDigest(nDigestPatterns int) (e error) {
	c.closeDigest()
	if nDigestPatterns == 0 {
		return nil
	}

	c.digestOpPool, e = cryptodev.NewOpPool(cryptodev.OpPoolConfig{
		Capacity: 2 * nDigestPatterns * (DigestLowWatermark + DigestBurstSize),
	}, c.socket())
	if e != nil {
		return e
	}

	drvPref := cryptodev.MultiSegDrv
	if DigestLinearize {
		drvPref = cryptodev.SingleSegDrv
	}
	c.digestCrypto, e = drvPref.Create(cryptodev.Config{
		NQueuePairs: nDigestPatterns,
	}, c.socket())
	if e != nil {
		return e
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
	ealthread.Launch(c.rx)
	ealthread.Launch(c.tx)
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
	c.closeDigest()
	must.Close(c.RxQueue())
	eal.Free(c.rxC)
	eal.Free(c.txC)
	return nil
}

func (c *Consumer) closeDigest() {
	if c.digestCrypto != nil {
		must.Close(c.digestCrypto)
		c.digestCrypto = nil
	}
	if c.digestOpPool != nil {
		must.Close(c.digestOpPool)
		c.digestOpPool = nil
	}
	for _, dp := range c.dPatterns {
		dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&dp.dataGen))
		dataGen.Close()
		eal.Free(dp)
	}
	c.dPatterns = nil
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

	c.rx = worker{
		Thread: ealthread.New(
			cptr.Func0.C(unsafe.Pointer(C.TgcRx_Run), unsafe.Pointer(c.rxC)),
			ealthread.InitStopFlag(unsafe.Pointer(&c.rxC.stop)),
		),
		loadStat: &c.rxC.loadStat,
	}
	c.tx = worker{
		Thread: ealthread.New(
			cptr.Func0.C(unsafe.Pointer(C.TgcTx_Run), unsafe.Pointer(c.txC)),
			ealthread.InitStopFlag(unsafe.Pointer(&c.txC.stop)),
		),
		loadStat: &c.txC.loadStat,
	}

	c.SetInterval(time.Millisecond)
	return c, nil
}
