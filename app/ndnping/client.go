package ndnping

/*
#include "client-rx.h"
#include "client-tx.h"
*/
import "C"
import (
	"errors"
	"math/rand"
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Client internal config.
const (
	Client_BurstSize        = C.PINGCLIENT_TX_BURST_SIZE
	Client_InterestLifetime = 1000
)

// Client instance and RX thread.
type Client struct {
	dpdk.ThreadBase
	c  *C.PingClientRx
	Tx ClientTxThread
}

func newClient(face iface.IFace, cfg ClientConfig) (client *Client) {
	socket := face.GetNumaSocket()
	crC := (*C.PingClientRx)(dpdk.Zmalloc("PingClientRx", C.sizeof_PingClientRx, socket))

	ctC := (*C.PingClientTx)(dpdk.Zmalloc("PingClientTx", C.sizeof_PingClientTx, socket))
	ctC.face = (C.FaceId)(face.GetFaceId())
	ctC.interestMbufHeadroom = C.uint16_t(appinit.SizeofEthLpHeaders() + ndn.EncodeInterest_GetHeadroom())
	ctC.interestMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_INT, socket).GetPtr())
	C.pcg32_srandom_r(&ctC.trafficRng, C.uint64_t(rand.Uint64()), C.uint64_t(time.Now().Unix()))
	C.NonceGen_Init(&ctC.nonceGen)

	client = new(Client)
	client.c = crC
	client.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&crC.stop))
	client.Tx.c = ctC
	client.Tx.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&ctC.stop))

	for _, pattern := range cfg.Patterns {
		client.AddPattern(pattern)
	}
	client.SetInterval(cfg.Interval)
	return client
}

func (client *Client) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(client.Tx.c.face))
}

func (client *Client) AddPattern(cfg ClientPattern) (index int, e error) {
	if client.c.nPatterns >= C.PINGCLIENT_MAX_PATTERNS {
		return -1, errors.New("too many patterns")
	}

	tpl := ndn.NewInterestTemplate()
	tpl.SetNamePrefix(cfg.Prefix)
	tpl.SetCanBePrefix(cfg.CanBePrefix)
	tpl.SetMustBeFresh(cfg.MustBeFresh)
	if cfg.InterestLifetime != time.Duration(0) {
		tpl.SetInterestLifetime(cfg.InterestLifetime)
	}
	if cfg.HopLimit != 0 {
		tpl.SetHopLimit(uint8(cfg.HopLimit))
	}

	index = int(client.c.nPatterns)
	client.clearCounter(index)
	rxP := &client.c.pattern[index]
	rxP.prefixLen = C.uint16_t(cfg.Prefix.Size())
	txP := &client.Tx.c.pattern[index]
	txP.seqNum.compT = C.TT_GenericNameComponent
	txP.seqNum.compL = C.uint8_t(C.sizeof_uint64_t)
	txP.seqNum.compV = C.uint64_t(rand.Uint64())
	if e = tpl.CopyToC(unsafe.Pointer(&txP.tpl),
		unsafe.Pointer(&txP.tplPrepareBuffer), int(unsafe.Sizeof(txP.tplPrepareBuffer)),
		unsafe.Pointer(&txP.prefixBuffer), int(unsafe.Sizeof(txP.prefixBuffer))); e != nil {
		return -1, e
	}

	client.c.nPatterns++
	client.Tx.c.nPatterns = client.c.nPatterns
	return index, nil
}

// Get average Interest interval.
func (client *Client) GetInterval() time.Duration {
	return dpdk.FromTscDuration(int64(client.Tx.c.burstInterval)) / Client_BurstSize
}

// Set average Interest interval.
// TX thread transmits Interests in bursts, so the specified interval will be converted to
// a burst interval with equivalent traffic amount.
func (client *Client) SetInterval(interval time.Duration) {
	client.Tx.c.burstInterval = C.TscDuration(dpdk.ToTscDuration(interval * Client_BurstSize))
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
