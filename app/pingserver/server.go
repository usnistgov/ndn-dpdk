package pingserver

/*
#include "server.h"
*/
import "C"
import (
	"fmt"
	"math/rand"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Server instance and thread.
type Server struct {
	dpdk.ThreadBase
	c      *C.PingServer
	seg1Mp dpdk.PktmbufPool
}

func New(face iface.IFace, cfg Config) (server *Server, e error) {
	socket := face.GetNumaSocket()
	serverC := (*C.PingServer)(dpdk.Zmalloc("PingServer", C.sizeof_PingServer, socket))
	serverC.dataMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_DATA0, socket).GetPtr())
	serverC.indirectMp = (*C.struct_rte_mempool)(appinit.MakePktmbufPool(
		appinit.MP_IND, socket).GetPtr())
	serverC.face = (C.FaceId)(face.GetFaceId())
	serverC.wantNackNoRoute = C.bool(cfg.Nack)
	C.pcg32_srandom_r(&serverC.replyRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))

	server = new(Server)
	server.seg1Mp = appinit.MakePktmbufPool(appinit.MP_DATA1, socket)
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

func (server *Server) AddPattern(cfg Pattern) (index int, e error) {
	if server.c.nPatterns >= C.PINGSERVER_MAX_PATTERNS {
		return -1, fmt.Errorf("cannot add more than %d patterns", C.PINGSERVER_MAX_PATTERNS)
	}
	if len(cfg.Replies) < 1 || len(cfg.Replies) > C.PINGSERVER_MAX_REPLIES {
		return -1, fmt.Errorf("must have between 1 and %d reply definitions", C.PINGSERVER_MAX_REPLIES)
	}

	index = int(server.c.nPatterns)
	patternC := &server.c.pattern[index]
	*patternC = C.PingServerPattern{}

	if e = cfg.Prefix.CopyToLName(unsafe.Pointer(&patternC.prefix), unsafe.Pointer(&patternC.prefixBuffer[0]), unsafe.Sizeof(patternC.prefixBuffer)); e != nil {
		return -1, e
	}

	for i, reply := range cfg.Replies {
		if reply.Weight < 1 {
			reply.Weight = 1
		}
		if patternC.nWeights+C.uint16_t(reply.Weight) >= C.PINGSERVER_MAX_SUM_WEIGHT {
			return -1, fmt.Errorf("sum of weight cannot exceed %d", C.PINGSERVER_MAX_SUM_WEIGHT)
		}
		for j := 0; j < reply.Weight; j++ {
			patternC.weight[patternC.nWeights] = C.PingReplyId(i)
			patternC.nWeights++
		}

		replyC := &patternC.reply[i]
		switch {
		case reply.Timeout:
			replyC.kind = C.PINGSERVER_REPLY_TIMEOUT
		case reply.Nack != ndn.NackReason_None:
			replyC.kind = C.PINGSERVER_REPLY_NACK
			replyC.nackReason = C.uint8_t(reply.Nack)
		default:
			replyC.kind = C.PINGSERVER_REPLY_DATA
			m, e := server.seg1Mp.Alloc()
			if e != nil {
				return -1, fmt.Errorf("cannot allocate from MP_DATA1 for reply definition %d", i)
			}
			dataGen := ndn.NewDataGen(m, reply.Suffix, reply.FreshnessPeriod, make(ndn.TlvBytes, reply.PayloadLen))
			replyC.dataGen = (*C.DataGen)(dataGen.GetPtr())
		}
	}
	patternC.nReplies = C.uint16_t(len(cfg.Replies))

	server.c.nPatterns++
	return index, nil
}

func (server *Server) SetRxQueue(queue dpdk.Ring) {
	server.c.rxQueue = (*C.struct_rte_ring)(queue.GetPtr())
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
