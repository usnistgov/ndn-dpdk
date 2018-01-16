package ndnping

/*
#include "client.h"
*/
import "C"
import (
	"fmt"
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/nameset"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type Client struct {
	c *C.NdnpingClient
}

func NewClient(face iface.Face) (client Client, e error) {
	socket := face.GetNumaSocket()
	client.c = (*C.NdnpingClient)(dpdk.Zmalloc("NdnpingClient", C.sizeof_NdnpingClient, socket))
	client.c.face = (*C.Face)(face.GetPtr())
	client.SetInterval(time.Second)

	client.c.mpInterest = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_INT, socket).GetPtr())

	C.NdnpingClient_Init(client.c)
	return client, nil
}

func (client Client) Close() error {
	client.getPatterns().Close()
	dpdk.Free(client.c)
	return nil
}

func (client Client) GetFace() iface.Face {
	return iface.FaceFromPtr(unsafe.Pointer(client.c.face))
}

func (client Client) getPatterns() nameset.NameSet {
	return nameset.FromPtr(unsafe.Pointer(&client.c.patterns))
}

func (client Client) AddPattern(comps ndn.TlvBytes, pct float32) {
	client.getPatterns().InsertWithZeroUsr(comps, int(C.sizeof_NdnpingClientPattern))
}

func (client Client) SetInterval(interval time.Duration) {
	client.c.interestInterval = C.double(float64(interval) / float64(time.Millisecond))
}

func (client Client) Run() int {
	return int(C.NdnpingClient_Run(client.c))
}

type ClientPatternCounters struct {
	NInterests uint64
	NData      uint64
	NNacks     uint64
}

func (cnt ClientPatternCounters) String() string {
	return fmt.Sprintf("%dI %dD(%0.2f%%) %dN(%0.2f%%)", cnt.NInterests,
		cnt.NData, float64(cnt.NData)/float64(cnt.NInterests)*100.0,
		cnt.NNacks, float64(cnt.NNacks)/float64(cnt.NInterests)*100.0)
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
	patterns := client.getPatterns()
	cnt.PerPattern = make([]ClientPatternCounters, patterns.Len())
	for i := 0; i < len(cnt.PerPattern); i++ {
		pattern := (*C.NdnpingClientPattern)(patterns.GetUsr(i))
		cnt.PerPattern[i].NInterests = uint64(pattern.nInterests)
		cnt.PerPattern[i].NData = uint64(pattern.nData)
		cnt.PerPattern[i].NNacks = uint64(pattern.nNacks)
		cnt.NInterests += uint64(pattern.nInterests)
		cnt.NData += uint64(pattern.nData)
		cnt.NNacks += uint64(pattern.nNacks)
	}

	cnt.NAllocError = uint64(client.c.nAllocError)
	return cnt
}
