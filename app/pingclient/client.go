package pingclient

/*
#include "rx.h"
#include "tx.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"math/rand"
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Client instance and RX thread.
type Client struct {
	dpdk.ThreadBase
	c  *C.PingClientRx
	Tx ClientTxThread
}

func New(face iface.IFace, cfg Config) (client *Client, e error) {
	socket := face.GetNumaSocket()
	crC := (*C.PingClientRx)(dpdk.Zmalloc("PingClientRx", C.sizeof_PingClientRx, socket))

	ctC := (*C.PingClientTx)(dpdk.Zmalloc("PingClientTx", C.sizeof_PingClientTx, socket))
	ctC.face = (C.FaceId)(face.GetFaceId())
	ctC.interestMbufHeadroom = C.uint16_t(appinit.SizeofEthLpHeaders() + ndn.EncodeInterest_GetHeadroom())
	ctC.interestMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_INT, socket).GetPtr())
	C.pcg32_srandom_r(&ctC.trafficRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))
	C.NonceGen_Init(&ctC.nonceGen)

	client = new(Client)
	client.c = crC
	client.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&crC.stop))
	client.Tx.c = ctC
	client.Tx.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&ctC.stop))

	for i, pattern := range cfg.Patterns {
		if _, e := client.AddPattern(pattern); e != nil {
			return nil, fmt.Errorf("pattern(%d): %s", i, e)
		}
	}
	client.SetInterval(cfg.Interval)
	return client, nil
}

func (client *Client) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(client.Tx.c.face))
}

func (client *Client) AddPattern(cfg Pattern) (index int, e error) {
	if client.c.nPatterns >= C.PINGCLIENT_MAX_PATTERNS {
		return -1, fmt.Errorf("cannot add more than %d patterns", C.PINGCLIENT_MAX_PATTERNS)
	}
	if cfg.Weight < 1 {
		cfg.Weight = 1
	}
	if client.Tx.c.nWeights+C.uint16_t(cfg.Weight) >= C.PINGCLIENT_MAX_SUM_WEIGHT {
		return -1, fmt.Errorf("sum of weight cannot exceed %d", C.PINGCLIENT_MAX_SUM_WEIGHT)
	}
	index = int(client.c.nPatterns)
	if cfg.SeqNumOffset != 0 && index == 0 {
		return -1, errors.New("first pattern cannot have SeqNumOffset")
	}

	tpl := ndn.NewInterestTemplate()
	tpl.SetNamePrefix(cfg.Prefix)
	tpl.SetCanBePrefix(cfg.CanBePrefix)
	tpl.SetMustBeFresh(cfg.MustBeFresh)
	if cfg.InterestLifetime != 0 {
		tpl.SetInterestLifetime(cfg.InterestLifetime)
	}
	if cfg.HopLimit != 0 {
		tpl.SetHopLimit(uint8(cfg.HopLimit))
	}

	client.clearCounter(index)
	rxP := &client.c.pattern[index]
	rxP.prefixLen = C.uint16_t(cfg.Prefix.Size())
	txP := &client.Tx.c.pattern[index]
	if e = tpl.CopyToC(unsafe.Pointer(&txP.tpl),
		unsafe.Pointer(&txP.tplPrepareBuffer), unsafe.Sizeof(txP.tplPrepareBuffer),
		unsafe.Pointer(&txP.prefixBuffer), unsafe.Sizeof(txP.prefixBuffer)); e != nil {
		return -1, e
	}
	txP.seqNum.compT = C.TT_GenericNameComponent
	txP.seqNum.compL = C.uint8_t(C.sizeof_uint64_t)
	txP.seqNum.compV = C.uint64_t(rand.Uint64())
	txP.seqNumOffset = C.uint32_t(cfg.SeqNumOffset)

	client.c.nPatterns++
	for i := 0; i < cfg.Weight; i++ {
		client.Tx.c.weight[client.Tx.c.nWeights] = C.PingPatternId(index)
		client.Tx.c.nWeights++
	}
	return index, nil
}

// Get average Interest interval.
func (client *Client) GetInterval() time.Duration {
	return dpdk.FromTscDuration(int64(client.Tx.c.burstInterval)) / C.PINGCLIENT_TX_BURST_SIZE
}

// Set average Interest interval.
// TX thread transmits Interests in bursts, so the specified interval will be converted to
// a burst interval with equivalent traffic amount.
func (client *Client) SetInterval(interval time.Duration) {
	client.Tx.c.burstInterval = C.TscDuration(dpdk.ToTscDuration(interval * C.PINGCLIENT_TX_BURST_SIZE))
}

func (client *Client) SetRxQueue(queue dpdk.Ring) {
	client.c.rxQueue = (*C.struct_rte_ring)(queue.GetPtr())
}

// Launch the RX thread.
func (client *Client) Launch() error {
	client.c.runNum++
	client.Tx.c.runNum = client.c.runNum
	return client.LaunchImpl(func() int {
		C.PingClientRx_Run(client.c)
		return 0
	})
}

// Stop the RX thread.
func (client *Client) Stop() error {
	return client.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&client.c.stop)))
}

// Close the client.
// Both RX and TX threads must be stopped before calling this.
func (client *Client) Close() error {
	dpdk.Free(client.c)
	dpdk.Free(client.Tx.c)
	return nil
}

// Client TX thread.
type ClientTxThread struct {
	dpdk.ThreadBase
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
	return tx.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&tx.c.stop)))
}

// No-op.
func (tx *ClientTxThread) Close() error {
	return nil
}
