// Package tgconsumer implements a traffic generator consumer.
package tgconsumer

/*
#include "../../csrc/tgconsumer/rx.h"
#include "../../csrc/tgconsumer/tx.h"

static_assert(offsetof(TgcTxPattern, digest) == offsetof(TgcTxPattern, seqNumOffset), "");
enum { c_offsetof_TgcTxPattern_DigestSeqNumOffset = offsetof(TgcTxPattern, digest) };
*/
import "C"
import (
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/tg/tgdef"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/core/pcg32"
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

type worker struct {
	ealthread.ThreadWithCtrl
}

var (
	_ ealthread.ThreadWithRole     = (*worker)(nil)
	_ ealthread.ThreadWithLoadStat = (*worker)(nil)
)

// ThreadRole implements ealthread.ThreadWithRole interface.
func (worker) ThreadRole() string {
	return tgdef.RoleConsumer
}

// Consumer represents a traffic generator consumer instance.
type Consumer struct {
	cfg    Config
	socket eal.NumaSocket
	rx     *worker
	tx     *worker
	rxC    *C.TgcRx
	txC    *C.TgcTx

	digestCrypto *cryptodev.CryptoDev
	dPatterns    []*C.TgcTxDigestPattern
}

var _ tgdef.Consumer = &Consumer{}

// Patterns returns traffic patterns.
func (c Consumer) Patterns() []Pattern {
	return c.cfg.Patterns
}

func (c *Consumer) initPatterns() (e error) {
	if e := c.prepareDigest(c.cfg.nDigestPatterns); e != nil {
		return fmt.Errorf("prepareDigest %w", e)
	}
	var dataGenVec pktmbuf.Vector
	if c.cfg.nDigestPatterns > 0 {
		payloadMp := ndni.PayloadMempool.Get(c.socket)
		dataGenVec, e = payloadMp.Alloc(c.cfg.nDigestPatterns)
		if e != nil {
			return e
		}
	}

	c.rxC.nPatterns = C.uint8_t(len(c.cfg.Patterns))
	c.txC.nWeights = C.uint32_t(c.cfg.nWeights)
	w := 0
	for i, pattern := range c.cfg.Patterns {
		c.assignPattern(i, pattern, dataGenVec.Take)

		for j := 0; j < pattern.Weight; j++ {
			c.txC.weight[w] = C.uint8_t(i)
			w++
		}

		c.clearCounter(i)
	}
	return nil
}

func (c *Consumer) assignPattern(i int, pattern Pattern, takeDataGenMbuf func() *pktmbuf.Packet) {
	rxP := &c.rxC.pattern[i]
	*rxP = C.TgcRxPattern{
		prefixLen: C.uint16_t(pattern.Prefix.Length()),
	}

	txP := &c.txC.pattern[i]
	*txP = C.TgcTxPattern{
		seqNumT: an.TtGenericNameComponent,
		seqNumL: C.sizeof_uint64_t,
		seqNumV: C.uint64_t(rand.Uint64()),
		digestT: an.TtImplicitSha256DigestComponent,
		digestL: ndni.ImplicitDigestLength,
	}
	pattern.InterestTemplateConfig.Apply(ndni.InterestTemplateFromPtr(unsafe.Pointer(&txP.tpl)))

	switch {
	case pattern.Digest != nil:
		c.assignDigestPattern(pattern, txP, takeDataGenMbuf)
	case pattern.SeqNumOffset != 0:
		txP.makeSuffix = C.TgcTxPattern_MakeSuffix(C.TgcTxPattern_MakeSuffix_Offset)
		*(*C.uint64_t)(unsafe.Add(unsafe.Pointer(txP), C.c_offsetof_TgcTxPattern_DigestSeqNumOffset)) = C.uint64_t(pattern.SeqNumOffset)
	default:
		txP.makeSuffix = C.TgcTxPattern_MakeSuffix(C.TgcTxPattern_MakeSuffix_Increment)
	}
}

func (c *Consumer) assignDigestPattern(pattern Pattern, txP *C.TgcTxPattern, takeDataGenMbuf func() *pktmbuf.Packet) {
	txP.makeSuffix = C.TgcTxPattern_MakeSuffix(C.TgcTxPattern_MakeSuffix_Digest)

	seqNumV := make([]byte, 8)
	binary.LittleEndian.PutUint64(seqNumV, uint64(txP.seqNumV))
	name := pattern.Prefix.Append(ndn.MakeNameComponent(an.TtGenericNameComponent, seqNumV))
	nameV, _ := tlv.EncodeValueOnly(name.Field())

	d := len(c.dPatterns)
	dp := eal.Zmalloc[C.TgcTxDigestPattern]("TgcTxDigestPattern", C.sizeof_TgcTxDigestPattern+len(nameV), c.socket)
	(*ndni.Mempools)(unsafe.Pointer(&dp.dataMp)).Assign(c.socket, ndni.DataMempool)
	c.digestCrypto.QueuePairs()[d].CopyToC(unsafe.Pointer(&dp.cqp))

	dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&dp.dataGen))
	pattern.Digest.Apply(dataGen, takeDataGenMbuf())

	nameVC := unsafe.Add(unsafe.Pointer(dp), C.sizeof_TgcTxDigestPattern)
	copy(unsafe.Slice((*byte)(nameVC), len(nameV)), nameV)
	dp.prefix.value = (*C.uint8_t)(nameVC)
	dp.prefix.length = C.uint16_t(len(nameV))

	*(**C.TgcTxDigestPattern)(unsafe.Add(unsafe.Pointer(txP), C.c_offsetof_TgcTxPattern_DigestSeqNumOffset)) = dp
	c.dPatterns = append(c.dPatterns, dp)
}

func (c *Consumer) prepareDigest(nDigestPatterns int) (e error) {
	c.closeDigest()
	if nDigestPatterns == 0 {
		return nil
	}

	var cfg cryptodev.VDevConfig
	cfg.NQueuePairs = nDigestPatterns
	cfg.Socket = c.socket
	c.digestCrypto, e = cryptodev.CreateVDev(cfg)
	if e != nil {
		return e
	}

	return nil
}

// Interval returns average Interest interval.
func (c Consumer) Interval() time.Duration {
	return eal.FromTscDuration(int64(c.txC.burstInterval)) / iface.MaxBurstSize
}

// Face returns the associated face.
func (c Consumer) Face() iface.Face {
	return iface.Get(iface.ID(c.txC.face))
}

func (c Consumer) rxQueue() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&c.rxC.rxQueue))
}

// ConnectRxQueues connects Data+Nack InputDemux to RxQueues.
func (c *Consumer) ConnectRxQueues(demuxD, demuxN *iface.InputDemux) {
	demuxD.InitFirst()
	demuxN.InitFirst()
	q := c.rxQueue()
	demuxD.SetDest(0, q)
	demuxN.SetDest(0, q)
}

// Workers returns worker threads.
func (c Consumer) Workers() []ealthread.ThreadWithRole {
	return []ealthread.ThreadWithRole{c.rx, c.tx}
}

// Launch launches RX and TX threads.
func (c *Consumer) Launch() {
	c.rxC.runNum++
	c.txC.runNum = c.rxC.runNum
	ealthread.Launch(c.rx)
	ealthread.Launch(c.tx)
}

// Stop stops RX and TX threads.
func (c *Consumer) Stop() error {
	return c.StopDelay(0)
}

// StopDelay stops the TX thread, sleep for the specified duration, then stops the RX thread.
func (c *Consumer) StopDelay(delay time.Duration) error {
	eTx := c.tx.Stop()
	time.Sleep(delay)
	eRx := c.rx.Stop()
	return multierr.Append(eTx, eRx)
}

// Close closes the consumer.
func (c *Consumer) Close() error {
	c.Stop()
	c.closeDigest()
	must.Close(c.rxQueue())
	eal.Free(c.rxC)
	eal.Free(c.txC)
	return nil
}

func (c *Consumer) closeDigest() {
	if c.digestCrypto != nil {
		must.Close(c.digestCrypto)
		c.digestCrypto = nil
	}
	for _, dp := range c.dPatterns {
		dataGen := ndni.DataGenFromPtr(unsafe.Pointer(&dp.dataGen))
		dataGen.Close()
		eal.Free(dp)
	}
	c.dPatterns = nil
}

// New creates a Consumer.
func New(face iface.Face, cfg Config) (c *Consumer, e error) {
	if e := cfg.Validate(); e != nil {
		return nil, e
	}

	socket := face.NumaSocket()
	c = &Consumer{
		cfg:    cfg,
		socket: socket,
		rxC:    eal.Zmalloc[C.TgcRx]("TgcRx", C.sizeof_TgcRx, socket),
		txC:    eal.Zmalloc[C.TgcTx]("TgcTx", C.sizeof_TgcTx, socket),
	}

	if e := c.rxQueue().Init(cfg.RxQueue, socket); e != nil {
		must.Close(c)
		return nil, fmt.Errorf("error initializing RxQueue %w", e)
	}

	c.txC.face = (C.FaceID)(face.ID())
	c.txC.interestMp = (*C.struct_rte_mempool)(ndni.InterestMempool.Get(socket).Ptr())
	pcg32.Init(unsafe.Pointer(&c.txC.trafficRng))
	pcg32.Init(unsafe.Pointer(&c.txC.nonceRng))

	c.rx = &worker{
		ThreadWithCtrl: ealthread.NewThreadWithCtrl(
			cptr.Func0.C(C.TgcRx_Run, c.rxC),
			unsafe.Pointer(&c.rxC.ctrl),
		),
	}
	c.tx = &worker{
		ThreadWithCtrl: ealthread.NewThreadWithCtrl(
			cptr.Func0.C(C.TgcTx_Run, c.txC),
			unsafe.Pointer(&c.txC.ctrl),
		),
	}

	c.txC.burstInterval = C.TscDuration(eal.ToTscDuration(
		cfg.Interval.DurationOr(nnduration.Nanoseconds(defaultInterval)) * iface.MaxBurstSize))
	if e := c.initPatterns(); e != nil {
		must.Close(c)
		return nil, fmt.Errorf("error setting patterns %w", e)
	}

	c.ClearCounters()
	return c, nil
}
