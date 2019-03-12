package ndnping

/*
#include "client.h"
#include "token.h"
*/
import "C"
import (
	"fmt"
	"math"
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/nameset"
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Client internal config.
const (
	Client_BurstSize        = C.NDNPINGCLIENT_TX_BURST_SIZE
	Client_InterestLifetime = 1000
)

// Client instance and RX thread.
type Client struct {
	dpdk.ThreadBase
	c  *C.NdnpingClient
	Tx *ClientTxThread
}

func newClient(face iface.IFace, cfg ClientConfig) (client *Client) {
	socket := face.GetNumaSocket()
	clientC := (*C.NdnpingClient)(dpdk.Zmalloc("NdnpingClient", C.sizeof_NdnpingClient, socket))
	clientC.face = (C.FaceId)(face.GetFaceId())

	clientC.interestMbufHeadroom = C.uint16_t(appinit.SizeofEthLpHeaders() + ndn.EncodeInterest_GetHeadroom())
	clientC.interestLifetime = C.uint16_t(Client_InterestLifetime)
	clientC.interestMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_INT, socket).GetPtr())

	C.NdnpingClient_Init(clientC)
	client = new(Client)
	client.c = clientC
	client.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&clientC.rxStop))
	client.Tx = new(ClientTxThread)
	client.Tx.c = clientC
	client.Tx.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&clientC.txStop))

	patterns := client.getPatterns()
	for _, patternCfg := range cfg.Patterns {
		patterns.InsertWithZeroUsr(patternCfg.Prefix, int(C.sizeof_NdnpingClientPattern))
	}

	client.SetInterval(cfg.Interval)
	return client
}

func (client *Client) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(client.c.face))
}

func (client *Client) getPatterns() nameset.NameSet {
	return nameset.FromPtr(unsafe.Pointer(&client.c.patterns))
}

// Get average Interest interval.
func (client *Client) GetInterval() time.Duration {
	return dpdk.FromTscDuration(int64(client.c.burstInterval)) / Client_BurstSize
}

// Set average Interest interval.
// TX thread transmits Interests in bursts, so the specified interval will be converted to
// a burst interval with equivalent traffic amount.
func (client *Client) SetInterval(interval time.Duration) {
	client.c.burstInterval = C.TscDuration(dpdk.ToTscDuration(interval * Client_BurstSize))
}

// Launch the RX thread.
func (client *Client) Launch() error {
	return client.LaunchImpl(func() int {
		C.NdnpingClient_RunRx(client.c)
		return 0
	})
}

// Stop the RX thread.
func (client *Client) Stop() error {
	return client.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&client.c.rxStop)))
}

// Close the client.
// Both RX and TX threads must be stopped before calling this.
func (client *Client) Close() error {
	client.getPatterns().Close()
	dpdk.Free(client.c)
	return nil
}

// Client TX thread.
type ClientTxThread struct {
	dpdk.ThreadBase
	c *C.NdnpingClient
}

// Launch the TX thread.
func (tx *ClientTxThread) Launch() error {
	return tx.LaunchImpl(func() int {
		C.NdnpingClient_RunTx(tx.c)
		return 0
	})
}

// Stop the TX thread.
func (tx *ClientTxThread) Stop() error {
	return tx.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&tx.c.txStop)))
}

// No-op.
func (tx *ClientTxThread) Close() error {
	return nil
}

type ClientPatternCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64

	NRttSamples uint64
	RttMin      time.Duration
	RttMax      time.Duration
	RttAvg      time.Duration
	RttStdev    time.Duration
}

func (cnt ClientPatternCounters) String() string {
	return fmt.Sprintf("%dI %dD(%0.2f%%) %dN(%0.2f%%) rtt=%0.3f/%0.3f/%0.3f/%0.3fms(%dsamp)",
		cnt.NInterests,
		cnt.NData, float64(cnt.NData)/float64(cnt.NInterests)*100.0,
		cnt.NNacks, float64(cnt.NNacks)/float64(cnt.NInterests)*100.0,
		float64(cnt.RttMin)/float64(time.Millisecond), float64(cnt.RttAvg)/float64(time.Millisecond),
		float64(cnt.RttMax)/float64(time.Millisecond), float64(cnt.RttStdev)/float64(time.Millisecond),
		cnt.NRttSamples)
}

type ClientCounters struct {
	PerPattern  []ClientPatternCounters
	NInterests  uint64
	NData       uint64
	NNacks      uint64
	NAllocError uint64
}

func (cnt ClientCounters) String() string {
	s := fmt.Sprintf("%dI %dD(%0.2f%%) %dN(%0.2f%%) %dalloc-error", cnt.NInterests,
		cnt.NData, float64(cnt.NData)/float64(cnt.NInterests)*100.0,
		cnt.NNacks, float64(cnt.NNacks)/float64(cnt.NInterests)*100.0,
		cnt.NAllocError)
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

func (client Client) ReadCounters() (cnt ClientCounters) {
	durationUnit := dpdk.GetNanosInTscUnit() *
		math.Pow(2.0, float64(C.NDNPING_TIMING_PRECISION))
	toDuration := func(d float64) time.Duration {
		return time.Duration(d * durationUnit)
	}

	patterns := client.getPatterns()
	cnt.PerPattern = make([]ClientPatternCounters, patterns.Len())
	for i := 0; i < len(cnt.PerPattern); i++ {
		pattern := (*C.NdnpingClientPattern)(patterns.GetUsr(i))
		rtt := running_stat.FromPtr(unsafe.Pointer(&pattern.rtt))
		perPattern := ClientPatternCounters{
			NInterests:  uint64(pattern.nInterests),
			NData:       uint64(pattern.nData),
			NNacks:      uint64(pattern.nNacks),
			NRttSamples: rtt.Len64(),
			RttMin:      toDuration(rtt.Min()),
			RttMax:      toDuration(rtt.Max()),
			RttAvg:      toDuration(rtt.Mean()),
			RttStdev:    toDuration(rtt.Stdev()),
		}
		cnt.PerPattern[i] = perPattern
		cnt.NInterests += perPattern.NInterests
		cnt.NData += perPattern.NData
		cnt.NNacks += perPattern.NNacks
	}

	cnt.NAllocError = uint64(client.c.nAllocError)
	return cnt
}
