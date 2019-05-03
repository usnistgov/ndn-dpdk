package ndnping

/*
#include "server.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"time"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Server instance and thread.
type Server struct {
	dpdk.ThreadBase
	c *C.PingServer
}

func newServer(face iface.IFace, cfg ServerConfig) (server *Server, e error) {
	socket := face.GetNumaSocket()
	serverC := (*C.PingServer)(dpdk.Zmalloc("PingServer", C.sizeof_PingServer, socket))
	serverC.dataMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA, socket).GetPtr())
	serverC.dataMbufHeadroom = C.uint16_t(appinit.SizeofEthLpHeaders() + ndn.EncodeData_GetHeadroom())
	serverC.face = (C.FaceId)(face.GetFaceId())
	serverC.wantNackNoRoute = C.bool(cfg.Nack)

	server = new(Server)
	server.c = serverC
	server.ResetThreadBase()
	dpdk.InitStopFlag(unsafe.Pointer(&serverC.stop))

	for i, pattern := range cfg.Patterns {
		if _, e := server.AddPattern(pattern); e != nil {
			return nil, fmt.Errorf("pattern(%d): %s", i, e)
		}
	}

	return server, nil
}

func (server *Server) AddPattern(cfg ServerPattern) (index int, e error) {
	if server.c.nPatterns >= C.PINGSERVER_MAX_PATTERNS {
		return -1, errors.New("too many patterns")
	}

	index = int(server.c.nPatterns)
	patternC := &server.c.pattern[index]
	*patternC = C.PingServerPattern{}

	if e = cfg.Prefix.CopyToLName(unsafe.Pointer(&patternC.prefix),
		unsafe.Pointer(&patternC.nameBuffer[0]), int(unsafe.Sizeof(patternC.nameBuffer))); e != nil {
		return -1, e
	}
	if cfg.Suffix != nil {
		nameBufferUsed := int(patternC.prefix.length)
		nameBufferFree := int(unsafe.Sizeof(patternC.nameBuffer)) - nameBufferUsed
		if e = cfg.Suffix.CopyToLName(unsafe.Pointer(&patternC.suffix),
			unsafe.Pointer(&patternC.nameBuffer[nameBufferUsed]), nameBufferFree); e != nil {
			return -1, e
		}
	}

	patternC.payloadL = C.uint16_t(cfg.PayloadLen)
	patternC.freshnessPeriod = C.uint32_t(cfg.FreshnessPeriod / time.Millisecond)

	server.c.nPatterns++
	return index, nil
}

// Launch the thread.
func (server *Server) Launch() error {
	return server.LaunchImpl(func() int {
		C.PingServer_Run(server.c)
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
	cnt.PerPattern = make([]ServerPatternCounters, int(server.c.nPatterns))
	for i := 0; i < int(server.c.nPatterns); i++ {
		pattern := server.c.pattern[i]
		cnt.PerPattern[i].NInterests = uint64(pattern.nInterests)
		cnt.NInterests += uint64(pattern.nInterests)
	}
	cnt.NNoMatch = uint64(server.c.nNoMatch)
	cnt.NAllocError = uint64(server.c.nAllocError)
	return cnt
}
