package ndnping

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

type Server struct {
	c *C.NdnpingServer
}

func NewServer(face iface.Face) (server Server, e error) {
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

func (server Server) Close() error {
	server.clearPayload()
	return nil
}

func (server Server) GetFace() iface.Face {
	return iface.FaceFromPtr(unsafe.Pointer(server.c.face))
}

func (server Server) SetNackNoRoute(enable bool) {
	server.c.wantNackNoRoute = C.bool(enable)
}

func (server Server) AddPrefix(comps ndn.TlvBytes) {
	prefixSet := nameset.FromPtr(unsafe.Pointer(&server.c.prefixes))
	prefixSet.Insert(comps)
}

func (server Server) clearPayload() {
	if server.c.payload != nil {
		dpdk.MbufFromPtr(unsafe.Pointer(server.c.payload)).Close()
	}
}

func (server Server) SetPayload(payload []byte) error {
	if len(payload) > Server_MaxPayloadSize {
		return fmt.Errorf("payload is too long")
	}

	server.clearPayload()

	numaSocket := server.GetFace().GetNumaSocket()
	mp := appinit.MakePktmbufPool(Server_PayloadMp, numaSocket)
	m, e := mp.Alloc()
	if e != nil {
		return fmt.Errorf("cannot allocate mbuf for payload: %v", e)
	}

	m.AsPacket().GetFirstSegment().AppendOctets(payload)
	server.c.payload = (*C.struct_rte_mbuf)(m.GetPtr())
	return nil
}

func (server Server) Run() int {
	C.NdnpingServer_Run(server.c)
	return 0
}

const Server_PayloadMp = "NdnpingServer_Payload"
const Server_MaxPayloadSize = 2048

func init() {
	appinit.RegisterMempool(Server_PayloadMp,
		appinit.MempoolConfig{
			Capacity:     15,
			CacheSize:    0,
			PrivSize:     0,
			DataRoomSize: Server_MaxPayloadSize,
		})
}
