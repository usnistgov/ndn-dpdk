package ndnping

/*
#include "client.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/nameset"
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
	client.c.runNum++
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
