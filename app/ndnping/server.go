package ndnping

/*
#include "server.h"
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

// Server internal config.
const (
	Server_BurstSize       = 64
	Server_FreshnessPeriod = 60000
)

type Server struct {
	c *C.NdnpingServer
}

func NewServer(face iface.IFace) (server Server, e error) {
	socket := face.GetNumaSocket()
	server.c = (*C.NdnpingServer)(dpdk.Zmalloc("NdnpingServer", C.sizeof_NdnpingServer, socket))
	server.c.face = (C.FaceId)(face.GetFaceId())
	server.c.freshnessPeriod = C.uint32_t(Server_FreshnessPeriod)

	server.c.dataMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA, socket).GetPtr())
	server.c.dataMbufHeadroom = C.uint16_t(appinit.SizeofEthLpHeaders() + ndn.EncodeData_GetHeadroom())

	return server, e
}

func (server Server) Close() error {
	server.SetNameSuffix(nil)
	server.SetPayloadLen(0)
	server.getPatterns().Close()
	dpdk.Free(server.c)
	return nil
}

func (server Server) GetFace() iface.IFace {
	return iface.Get(iface.FaceId(server.c.face))
}

func (server Server) SetNackNoRoute(enable bool) {
	server.c.wantNackNoRoute = C.bool(enable)
}

func (server Server) SetNameSuffix(n *ndn.Name) {
	if server.c.nameSuffix.value != nil {
		dpdk.Free(server.c.nameSuffix.value)
		server.c.nameSuffix.value = nil
	}
	if len := n.Size(); len > 0 {
		v := uintptr(dpdk.Zmalloc("NdnpingServerSuffix", len, dpdk.NUMA_SOCKET_ANY))
		for i, ch := range n.GetValue() {
			*(*byte)(unsafe.Pointer(v + uintptr(i))) = ch
		}
		server.c.nameSuffix.value = (*C.uint8_t)(unsafe.Pointer(v))
		server.c.nameSuffix.length = (C.uint16_t)(len)
	}
}

func (server Server) SetFreshnessPeriod(freshness time.Duration) {
	server.c.freshnessPeriod = C.uint32_t(freshness / time.Millisecond)
}

func (server Server) SetPayloadLen(len int) {
	if server.c.payloadV != nil {
		dpdk.Free(server.c.payloadV)
	}
	if len > 0 {
		server.c.payloadV = (*C.uint8_t)(dpdk.Zmalloc("NdnpingServerPayload", len, dpdk.NUMA_SOCKET_ANY))
		server.c.payloadL = C.uint16_t(len)
	}
}

func (server Server) getPatterns() nameset.NameSet {
	return nameset.FromPtr(unsafe.Pointer(&server.c.patterns))
}

func (server Server) AddPattern(name *ndn.Name) {
	server.getPatterns().InsertWithZeroUsr(name, int(C.sizeof_NdnpingServerPattern))
}

func (server Server) Run() int {
	face := server.GetFace()
	appinit.MakeRxLooper(face).RxLoop(Server_BurstSize,
		unsafe.Pointer(C.NdnpingServer_Rx), unsafe.Pointer(server.c))
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
