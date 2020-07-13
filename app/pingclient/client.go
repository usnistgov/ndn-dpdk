package pingclient

/*
#include "../../csrc/pingclient/rx.h"
#include "../../csrc/pingclient/tx.h"
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
)

// Client instance and RX thread.
type Client struct {
	Rx ealthread.Thread
	Tx ealthread.Thread

	rxC *C.PingClientRx
	txC *C.PingClientTx
}

func New(face iface.Face, cfg Config) (*Client, error) {
	socket := face.NumaSocket()
	rxC := (*C.PingClientRx)(eal.Zmalloc("PingClientRx", C.sizeof_PingClientRx, socket))
	cfg.RxQueue.DisableCoDel = true
	if e := iface.PktQueueFromPtr(unsafe.Pointer(&rxC.rxQueue)).Init(cfg.RxQueue, socket); e != nil {
		eal.Free(rxC)
		return nil, nil
	}

	txC := (*C.PingClientTx)(eal.Zmalloc("PingClientTx", C.sizeof_PingClientTx, socket))
	txC.face = (C.FaceID)(face.ID())
	txC.interestMp = (*C.struct_rte_mempool)(ndni.InterestMempool.MakePool(socket).Ptr())
	C.pcg32_srandom_r(&txC.trafficRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	C.NonceGen_Init(&txC.nonceGen)

	var client Client
	client.rxC = rxC
	client.txC = txC
	client.Rx = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.PingClientRx_Run), unsafe.Pointer(rxC)),
		ealthread.InitStopFlag(unsafe.Pointer(&rxC.stop)),
	)
	client.Tx = ealthread.New(
		cptr.Func0.C(unsafe.Pointer(C.PingClientTx_Run), unsafe.Pointer(txC)),
		ealthread.InitStopFlag(unsafe.Pointer(&txC.stop)),
	)

	for i, pattern := range cfg.Patterns {
		if _, e := client.AddPattern(pattern); e != nil {
			return nil, fmt.Errorf("pattern(%d): %s", i, e)
		}
	}
	client.SetInterval(cfg.Interval.Duration())
	return &client, nil
}

func (client *Client) GetFace() iface.Face {
	return iface.Get(iface.ID(client.txC.face))
}

func (client *Client) AddPattern(cfg Pattern) (index int, e error) {
	if client.rxC.nPatterns >= C.PINGCLIENT_MAX_PATTERNS {
		return -1, fmt.Errorf("cannot add more than %d patterns", C.PINGCLIENT_MAX_PATTERNS)
	}
	if cfg.Weight < 1 {
		cfg.Weight = 1
	}
	if client.txC.nWeights+C.uint16_t(cfg.Weight) >= C.PINGCLIENT_MAX_SUM_WEIGHT {
		return -1, fmt.Errorf("sum of weight cannot exceed %d", C.PINGCLIENT_MAX_SUM_WEIGHT)
	}
	index = int(client.rxC.nPatterns)
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

	client.clearCounter(index)
	rxP := &client.rxC.pattern[index]
	rxP.prefixLen = C.uint16_t(cfg.Prefix.Length())
	txP := &client.txC.pattern[index]
	ndni.InterestTemplateFromPtr(unsafe.Pointer(&txP.tpl)).Init(tplArgs...)

	txP.seqNum.compT = C.TtGenericNameComponent
	txP.seqNum.compL = C.uint8_t(C.sizeof_uint64_t)
	txP.seqNum.compV = C.uint64_t(rand.Uint64())
	txP.seqNumOffset = C.uint32_t(cfg.SeqNumOffset)

	client.rxC.nPatterns++
	for i := 0; i < cfg.Weight; i++ {
		client.txC.weight[client.txC.nWeights] = C.PingPatternId(index)
		client.txC.nWeights++
	}
	return index, nil
}

// Get average Interest interval.
func (client *Client) GetInterval() time.Duration {
	return eal.FromTscDuration(int64(client.txC.burstInterval)) / C.PINGCLIENT_TX_BURST_SIZE
}

// Set average Interest interval.
// TX thread transmits Interests in bursts, so the specified interval will be converted to
// a burst interval with equivalent traffic amount.
func (client *Client) SetInterval(interval time.Duration) {
	client.txC.burstInterval = C.TscDuration(eal.ToTscDuration(interval * C.PINGCLIENT_TX_BURST_SIZE))
}

func (client *Client) RxQueue() *iface.PktQueue {
	return iface.PktQueueFromPtr(unsafe.Pointer(&client.rxC.rxQueue))
}

func (client *Client) SetLCores(rxLCore, txLCore eal.LCore) {
	client.Rx.SetLCore(rxLCore)
	client.Tx.SetLCore(txLCore)
}

// Launch RX and TX threads.
func (client *Client) Launch() {
	client.rxC.runNum++
	client.txC.runNum = client.rxC.runNum
	client.Rx.Launch()
	client.Tx.Launch()
}

// Stop RX and TX threads.
func (client *Client) Stop(delay time.Duration) error {
	eTx := client.Tx.Stop()
	time.Sleep(delay)
	eRx := client.Rx.Stop()
	if eRx != nil || eTx != nil {
		return fmt.Errorf("RX %v; TX %v", eRx, eTx)
	}
	return nil
}

// Close the client.
// Both RX and TX threads must be stopped before calling this.
func (client *Client) Close() error {
	client.RxQueue().Close()
	eal.Free(client.rxC)
	eal.Free(client.txC)
	return nil
}
