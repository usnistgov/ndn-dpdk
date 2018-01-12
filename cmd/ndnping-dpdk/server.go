package main

/*
#include "server.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/nameset"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

type NdnpingServer struct {
	c *C.NdnpingServer
}

func NewNdnpingServer(face iface.Face) (server NdnpingServer, e error) {
	server.c = new(C.NdnpingServer)
	server.c.face = (*C.Face)(face.GetPtr())
	e = server.SetPayload([]byte{})

	numaSocket := face.GetNumaSocket()
	server.c.mpData1 = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA1, numaSocket).GetPtr())
	server.c.mpData2 = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA2, numaSocket).GetPtr())
	server.c.mpIndirect = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_IND, numaSocket).GetPtr())

	return server, e
}

func (server NdnpingServer) GetFace() iface.Face {
	return iface.FaceFromPtr(unsafe.Pointer(server.c.face))
}

func (server NdnpingServer) SetNackNoRoute(enable bool) {
	server.c.wantNackNoRoute = C.bool(enable)
}

func (server NdnpingServer) AddPrefix(comps ndn.TlvBytes) {
	prefixSet := nameset.FromPtr(unsafe.Pointer(&server.c.prefixes))
	prefixSet.Insert(comps)
}

func (server NdnpingServer) SetPayload(payload []byte) error {
	if len(payload) > NdnpingServer_MaxPayloadSize {
		return fmt.Errorf("payload is too long")
	}

	if server.c.payload != nil {
		dpdk.MbufFromPtr(unsafe.Pointer(server.c.payload)).Close()
	}

	numaSocket := iface.FaceFromPtr(unsafe.Pointer(server.c.face)).GetNumaSocket()
	mp := appinit.MakePktmbufPool(ndnpingServer_PayloadMp, numaSocket)
	m, e := mp.Alloc()
	if e != nil {
		return fmt.Errorf("cannot allocate mbuf for payload: %v", e)
	}

	m.AsPacket().GetFirstSegment().AppendOctets(payload)
	server.c.payload = (*C.struct_rte_mbuf)(m.GetPtr())
	return nil
}

func (server NdnpingServer) Run() int {
	return int(C.NdnpingServer_Run(server.c))
}

const ndnpingServer_PayloadMp = "NdnpingServer_Payload"
const NdnpingServer_MaxPayloadSize = 2048

func init() {
	appinit.RegisterMempool(ndnpingServer_PayloadMp,
		appinit.MempoolConfig{
			Capacity:     15,
			CacheSize:    0,
			PrivSize:     0,
			DataRoomSize: NdnpingServer_MaxPayloadSize,
		})
}
