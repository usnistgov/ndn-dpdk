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
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type Client struct {
	c *C.NdnpingClient
}

func NewClient(face iface.Face) (client Client, e error) {
	client.c = (*C.NdnpingClient)(C.calloc(1, C.sizeof_NdnpingClient))
	client.c.face = (*C.Face)(face.GetPtr())
	client.SetInterval(time.Second)

	numaSocket := face.GetNumaSocket()
	client.c.mpInterest = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_INT, numaSocket).GetPtr())

	C.NdnpingClient_Init(client.c)
	return client, nil
}

func (client Client) Close() error {
	nameset.FromPtr(unsafe.Pointer(&client.c.prefixes)).Close()
	C.free(unsafe.Pointer(client.c))
	return nil
}

func (client Client) GetFace() iface.Face {
	return iface.FaceFromPtr(unsafe.Pointer(client.c.face))
}

func (client Client) AddPattern(comps ndn.TlvBytes, pct float32) {
	prefixSet := nameset.FromPtr(unsafe.Pointer(&client.c.prefixes))
	prefixSet.Insert(comps)
}

func (client Client) SetInterval(interval time.Duration) {
	client.c.interestInterval = C.double(float64(interval) / float64(time.Millisecond))
}

func (client Client) Run() int {
	return int(C.NdnpingClient_Run(client.c))
}
