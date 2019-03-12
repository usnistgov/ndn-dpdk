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

// Server internal config.
const (
	Server_BurstSize       = C.NDNPINGSERVER_BURST_SIZE
	Server_FreshnessPeriod = 60000
)

// Server instance and thread.
type Server struct {
	dpdk.ThreadBase
	c *C.NdnpingServer
}

func newServer(face iface.IFace, cfg ServerConfig) (server *Server) {
	socket := face.GetNumaSocket()
	serverC := (*C.NdnpingServer)(dpdk.Zmalloc("NdnpingServer", C.sizeof_NdnpingServer, socket))
	serverC.face = (C.FaceId)(face.GetFaceId())
	serverC.freshnessPeriod = C.uint32_t(Server_FreshnessPeriod)

	serverC.dataMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA, socket).GetPtr())
	serverC.dataMbufHeadroom = C.uint16_t(appinit.SizeofEthLpHeaders() + ndn.EncodeData_GetHeadroom())

	server = new(Server)
	server.c = serverC
	server.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&serverC.stop))

	for _, patternCfg := range cfg.Patterns {
		server.addPattern(patternCfg)
	}
	serverC.wantNackNoRoute = C.bool(cfg.Nack)

	return server
}

func (server *Server) getPatterns() nameset.NameSet {
	return nameset.FromPtr(unsafe.Pointer(&server.c.patterns))
}

func (server *Server) addPattern(cfg ServerPattern) {
	suffixL := 0
	if cfg.Suffix != nil {
		suffixL = cfg.Suffix.Size()
	}
	sizeofUsr := int(C.sizeof_NdnpingServerPattern) + suffixL

	_, usr := server.getPatterns().InsertWithZeroUsr(cfg.Prefix, sizeofUsr)
	patternC := (*C.NdnpingServerPattern)(usr)
	patternC.payloadL = C.uint16_t(cfg.PayloadLen)
	if suffixL > 0 {
		suffixV := unsafe.Pointer(uintptr(usr) + uintptr(C.sizeof_NdnpingServerPattern))
		oldSuffixV := cfg.Suffix.GetValue()
		C.memcpy(suffixV, unsafe.Pointer(&oldSuffixV[0]), C.size_t(suffixL))
		patternC.nameSuffix.value = (*C.uint8_t)(suffixV)
		patternC.nameSuffix.length = (C.uint16_t)(suffixL)
	}
}

// Launch the thread.
func (server *Server) Launch() error {
	return server.LaunchImpl(func() int {
		C.NdnpingServer_Run(server.c)
		return 0
	})
}

// Stop the thread.
func (server *Server) Stop() error {
	return server.StopImpl(dpdk.NewStopFlag(unsafe.Pointer(&server.c.stop)))
}

// Close the server.
// The thread must be stopped before calling this.
func (server *Server) Close() error {
	server.getPatterns().Close()
	dpdk.Free(server.c)
	return nil
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

func (server *Server) ReadCounters() (cnt ServerCounters) {
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
