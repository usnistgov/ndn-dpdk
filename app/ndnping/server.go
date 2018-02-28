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
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/ndn"
)

type Server struct {
	c *C.NdnpingServer
}

func NewServer(face iface.Face) (server Server, e error) {
	socket := face.GetNumaSocket()
	server.c = (*C.NdnpingServer)(dpdk.Zmalloc("NdnpingServer", C.sizeof_NdnpingServer, socket))
	server.c.face = (*C.Face)(face.GetPtr())
	e = server.SetPayload([]byte{})

	server.c.data1Mp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA1, socket).GetPtr())
	server.c.data2Mp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA2, socket).GetPtr())
	server.c.indirectMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_IND, socket).GetPtr())

	return server, e
}

func (server Server) Close() error {
	server.getPatterns().Close()
	server.clearPayload()
	dpdk.Free(server.c)
	return nil
}

func (server Server) GetFace() iface.Face {
	return iface.FaceFromPtr(unsafe.Pointer(server.c.face))
}

func (server Server) SetNackNoRoute(enable bool) {
	server.c.wantNackNoRoute = C.bool(enable)
}

func (server Server) getPatterns() nameset.NameSet {
	return nameset.FromPtr(unsafe.Pointer(&server.c.patterns))
}

func (server Server) AddPattern(name *ndn.Name) {
	server.getPatterns().InsertWithZeroUsr(name, int(C.sizeof_NdnpingServerPattern))
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

	m.AsPacket().GetFirstSegment().Append(payload)
	server.c.payload = (*C.struct_rte_mbuf)(m.GetPtr())
	return nil
}

const Server_PayloadMp = "NdnpingServer_Payload"
const Server_MaxPayloadSize = 2048
const Server_BurstSize = 64

func init() {
	appinit.RegisterMempool(Server_PayloadMp,
		appinit.MempoolConfig{
			Capacity:     15,
			CacheSize:    0,
			PrivSize:     0,
			DataRoomSize: Server_MaxPayloadSize,
		})
}

func (server Server) Run() int {
	face := server.GetFace()
	if face.GetFaceId().GetKind() == iface.FaceKind_EthDev {
		ethface.EthFace{face}.RxLoop(Server_BurstSize, unsafe.Pointer(C.NdnpingServer_Rx),
			unsafe.Pointer(server.c))
	} else {
		C.NdnpingServer_Run(server.c)
	}
	return 0
}

type ServerPatternCounters struct {
	NInterests uint64
}

func (cnt ServerPatternCounters) String() string {
	return fmt.Sprintf("%dI", cnt.NInterests)
}

type ServerCounters struct {
	PerPattern  []ServerPatternCounters
	NInterests  uint64
	NNoMatch    uint64
	NAllocError uint64
}

func (cnt ServerCounters) String() string {
	s := fmt.Sprintf("%dI %dno-match %dalloc-error", cnt.NInterests, cnt.NNoMatch, cnt.NAllocError)
	for i, pcnt := range cnt.PerPattern {
		s += fmt.Sprintf(", pattern(%d) %s", i, pcnt)
	}
	return s
}

func (server Server) ReadCounters() (cnt ServerCounters) {
	patterns := server.getPatterns()
	cnt.PerPattern = make([]ServerPatternCounters, patterns.Len())
	for i := 0; i < len(cnt.PerPattern); i++ {
		pattern := (*C.NdnpingServerPattern)(patterns.GetUsr(i))
		cnt.PerPattern[i].NInterests = uint64(pattern.nInterests)
		cnt.NInterests += uint64(pattern.nInterests)
	}

	cnt.NNoMatch = uint64(server.c.nNoMatch)
	cnt.NAllocError = uint64(server.c.nAllocError)
	return cnt
}
