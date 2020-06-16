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

	"github.com/usnistgov/ndn-dpdk/app/ping/pingmempool"
	"github.com/usnistgov/ndn-dpdk/container/pktqueue"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Client instance and RX thread.
type Client struct {
	Rx ClientRxThread
	Tx ClientTxThread
}

func New(face iface.IFace, cfg Config) (client *Client, e error) {
	socket := face.GetNumaSocket()
	crC := (*C.PingClientRx)(eal.Zmalloc("PingClientRx", C.sizeof_PingClientRx, socket))
	cfg.RxQueue.DisableCoDel = true
	if _, e := pktqueue.NewAt(unsafe.Pointer(&crC.rxQueue), cfg.RxQueue, fmt.Sprintf("PingClient%d_rxQ", face.GetFaceId()), socket); e != nil {
		eal.Free(crC)
		return nil, nil
	}

	ctC := (*C.PingClientTx)(eal.Zmalloc("PingClientTx", C.sizeof_PingClientTx, socket))
	ctC.face = (C.FaceId)(face.GetFaceId())
	ctC.interestMp = (*C.struct_rte_mempool)(pingmempool.Interest.MakePool(socket).GetPtr())
	C.pcg32_srandom_r(&ctC.trafficRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	C.NonceGen_Init(&ctC.nonceGen)

	client = new(Client)
	client.Rx.c = crC
	eal.InitStopFlag(unsafe.Pointer(&crC.stop))
	client.Tx.c = ctC
	eal.InitStopFlag(unsafe.Pointer(&ctC.stop))

	for i, pattern := range cfg.Patterns {
		if _, e := client.AddPattern(pattern); e != nil {
			return nil, fmt.Errorf("pattern(%d): %s", i, e)
		}
	}
	client.SetInterval(cfg.Interval.Duration())
	return client, nil
}

func (client *Client) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(client.Tx.c.face))
}

func (client *Client) AddPattern(cfg Pattern) (index int, e error) {
	if client.Rx.c.nPatterns >= C.PINGCLIENT_MAX_PATTERNS {
		return -1, fmt.Errorf("cannot add more than %d patterns", C.PINGCLIENT_MAX_PATTERNS)
	}
	if cfg.Weight < 1 {
		cfg.Weight = 1
	}
	if client.Tx.c.nWeights+C.uint16_t(cfg.Weight) >= C.PINGCLIENT_MAX_SUM_WEIGHT {
		return -1, fmt.Errorf("sum of weight cannot exceed %d", C.PINGCLIENT_MAX_SUM_WEIGHT)
	}
	index = int(client.Rx.c.nPatterns)
	if cfg.SeqNumOffset != 0 && index == 0 {
		return -1, errors.New("first pattern cannot have SeqNumOffset")
	}

	tplArgs := []interface{}{cfg.Prefix}
	if cfg.CanBePrefix {
		tplArgs = append(tplArgs, ndni.CanBePrefixFlag)
	}
	if cfg.MustBeFresh {
		tplArgs = append(tplArgs, ndni.MustBeFreshFlag)
	}
	if lifetime := cfg.InterestLifetime.Duration(); lifetime != 0 {
		tplArgs = append(tplArgs, lifetime)
	}
	if cfg.HopLimit != 0 {
		tplArgs = append(tplArgs, uint8(cfg.HopLimit))
	}

	client.clearCounter(index)
	rxP := &client.Rx.c.pattern[index]
	rxP.prefixLen = C.uint16_t(cfg.Prefix.Size())
	txP := &client.Tx.c.pattern[index]
	if e = ndni.InterestTemplateFromPtr(unsafe.Pointer(&txP.tpl)).Init(tplArgs...); e != nil {
		return -1, e
	}
	txP.seqNum.compT = C.TtGenericNameComponent
	txP.seqNum.compL = C.uint8_t(C.sizeof_uint64_t)
	txP.seqNum.compV = C.uint64_t(rand.Uint64())
	txP.seqNumOffset = C.uint32_t(cfg.SeqNumOffset)

	client.Rx.c.nPatterns++
	for i := 0; i < cfg.Weight; i++ {
		client.Tx.c.weight[client.Tx.c.nWeights] = C.PingPatternId(index)
		client.Tx.c.nWeights++
	}
	return index, nil
}

// Get average Interest interval.
func (client *Client) GetInterval() time.Duration {
	return eal.FromTscDuration(int64(client.Tx.c.burstInterval)) / C.PINGCLIENT_TX_BURST_SIZE
}

// Set average Interest interval.
// TX thread transmits Interests in bursts, so the specified interval will be converted to
// a burst interval with equivalent traffic amount.
func (client *Client) SetInterval(interval time.Duration) {
	client.Tx.c.burstInterval = C.TscDuration(eal.ToTscDuration(interval * C.PINGCLIENT_TX_BURST_SIZE))
}

func (client *Client) GetRxQueue() *pktqueue.PktQueue {
	return pktqueue.FromPtr(unsafe.Pointer(&client.Rx.c.rxQueue))
}

func (client *Client) SetLCores(rxLCore, txLCore eal.LCore) {
	client.Rx.SetLCore(rxLCore)
	client.Tx.SetLCore(txLCore)
}

// Launch RX and TX threads.
func (client *Client) Launch() error {
	client.Rx.c.runNum++
	client.Tx.c.runNum = client.Rx.c.runNum
	eRx := client.Rx.Launch()
	eTx := client.Tx.Launch()
	if eRx != nil || eTx != nil {
		return fmt.Errorf("RX %v; TX %v", eRx, eTx)
	}
	return nil
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
	client.GetRxQueue().Close()
	eal.Free(client.Rx.c)
	eal.Free(client.Tx.c)
	return nil
}

// Client RX thread.
type ClientRxThread struct {
	eal.ThreadBase
	c *C.PingClientRx
}

// Launch the RX thread.
func (rx *ClientRxThread) Launch() error {
	return rx.LaunchImpl(func() int {
		C.PingClientRx_Run(rx.c)
		return 0
	})
}

// Stop the RX thread.
func (rx *ClientRxThread) Stop() error {
	return rx.StopImpl(eal.NewStopFlag(unsafe.Pointer(&rx.c.stop)))
}

// No-op.
func (rx *ClientRxThread) Close() error {
	return nil
}

// Client TX thread.
type ClientTxThread struct {
	eal.ThreadBase
	c *C.PingClientTx
}

// Launch the TX thread.
func (tx *ClientTxThread) Launch() error {
	return tx.LaunchImpl(func() int {
		C.PingClientTx_Run(tx.c)
		return 0
	})
}

// Stop the TX thread.
func (tx *ClientTxThread) Stop() error {
	return tx.StopImpl(eal.NewStopFlag(unsafe.Pointer(&tx.c.stop)))
}

// No-op.
func (tx *ClientTxThread) Close() error {
	return nil
}
