package pingserver

/*
#include "../../csrc/pingserver/server.h"
*/
import "C"
import (
	"fmt"
	"math/rand"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/app/ping/pingmempool"
	"github.com/usnistgov/ndn-dpdk/container/pktqueue"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Server instance and thread.
type Server struct {
	ealthread.Thread
	c      *C.PingServer
	seg1Mp *pktmbuf.Pool
}

func New(face iface.Face, index int, cfg Config) (*Server, error) {
	faceID := face.ID()
	socket := face.NumaSocket()
	serverC := (*C.PingServer)(eal.Zmalloc("PingServer", C.sizeof_PingServer, socket))

	cfg.RxQueue.DisableCoDel = true
	if _, e := pktqueue.NewAt(unsafe.Pointer(&serverC.rxQueue), cfg.RxQueue, fmt.Sprintf("PingServer%d-%d_rxQ", faceID, index), socket); e != nil {
		eal.Free(serverC)
		return nil, nil
	}

	serverC.dataMp = (*C.struct_rte_mempool)(pingmempool.Data.MakePool(socket).Ptr())
	serverC.indirectMp = (*C.struct_rte_mempool)(pktmbuf.Indirect.MakePool(socket).Ptr())
	serverC.face = (C.FaceID)(faceID)
	serverC.wantNackNoRoute = C.bool(cfg.Nack)
	C.pcg32_srandom_r(&serverC.replyRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))

	server := &Server{
		seg1Mp: pingmempool.Payload.MakePool(socket),
		c:      serverC,
	}
	server.Thread = ealthread.New(
		cptr.CFunction(unsafe.Pointer(C.PingServer_Run), unsafe.Pointer(server.c)),
		ealthread.InitStopFlag(unsafe.Pointer(&serverC.stop)),
	)

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

	prefixV, _ := cfg.Prefix.MarshalBinary()
	if len(prefixV) > len(patternC.prefixBuffer) {
		return -1, fmt.Errorf("prefix too long")
	}
	for i, b := range prefixV {
		patternC.prefixBuffer[i] = C.uint8_t(b)
	}
	patternC.prefix.value = &patternC.prefixBuffer[0]
	patternC.prefix.length = C.uint16_t(len(prefixV))

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
		case reply.Nack != an.NackNone:
			replyC.kind = C.PINGSERVER_REPLY_NACK
			replyC.nackReason = C.uint8_t(reply.Nack)
		default:
			replyC.kind = C.PINGSERVER_REPLY_DATA
			vec, e := server.seg1Mp.Alloc(1)
			if e != nil {
				return -1, fmt.Errorf("cannot allocate from MP_DATA1 for reply definition %d", i)
			}
			dataGen := ndni.NewDataGen(vec[0], reply.Suffix, reply.FreshnessPeriod.Duration(), make([]byte, reply.PayloadLen))
			replyC.dataGen = (*C.DataGen)(dataGen.Ptr())
		}
	}
	patternC.nReplies = C.uint16_t(len(cfg.Replies))

	server.c.nPatterns++
	return index, nil
}

func (server *Server) GetRxQueue() *pktqueue.PktQueue {
	return pktqueue.FromPtr(unsafe.Pointer(&server.c.rxQueue))
}

// Close the server.
// The thread must be stopped before calling this.
func (server *Server) Close() error {
	server.Stop()
	server.GetRxQueue().Close()
	eal.Free(server.c)
	return nil
}
