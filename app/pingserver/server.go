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
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/an"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Server instance and thread.
type Server struct {
	eal.ThreadBase
	c      *C.PingServer
	seg1Mp *pktmbuf.Pool
}

func New(face iface.IFace, index int, cfg Config) (server *Server, e error) {
	faceId := face.GetFaceId()
	socket := face.GetNumaSocket()
	serverC := (*C.PingServer)(eal.Zmalloc("PingServer", C.sizeof_PingServer, socket))

	cfg.RxQueue.DisableCoDel = true
	if _, e := pktqueue.NewAt(unsafe.Pointer(&serverC.rxQueue), cfg.RxQueue, fmt.Sprintf("PingServer%d-%d_rxQ", faceId, index), socket); e != nil {
		eal.Free(serverC)
		return nil, nil
	}

	serverC.dataMp = (*C.struct_rte_mempool)(pingmempool.Data.MakePool(socket).GetPtr())
	serverC.indirectMp = (*C.struct_rte_mempool)(pktmbuf.Indirect.MakePool(socket).GetPtr())
	serverC.face = (C.FaceId)(faceId)
	serverC.wantNackNoRoute = C.bool(cfg.Nack)
	C.pcg32_srandom_r(&serverC.replyRng, C.uint64_t(rand.Uint64()), C.uint64_t(rand.Uint64()))

	server = new(Server)
	server.seg1Mp = pingmempool.Payload.MakePool(socket)
	server.c = serverC
	eal.InitStopFlag(unsafe.Pointer(&serverC.stop))

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
		case reply.Nack != an.NackNone:
			replyC.kind = C.PINGSERVER_REPLY_NACK
			replyC.nackReason = C.uint8_t(reply.Nack)
		default:
			replyC.kind = C.PINGSERVER_REPLY_DATA
			vec, e := server.seg1Mp.Alloc(1)
			if e != nil {
				return -1, fmt.Errorf("cannot allocate from MP_DATA1 for reply definition %d", i)
			}
			dataGen := ndni.NewDataGen(vec[0], reply.Suffix, reply.FreshnessPeriod.Duration(), make(ndni.TlvBytes, reply.PayloadLen))
			replyC.dataGen = (*C.DataGen)(dataGen.GetPtr())
		}
	}
	patternC.nReplies = C.uint16_t(len(cfg.Replies))

	server.c.nPatterns++
	return index, nil
}

func (server *Server) GetRxQueue() *pktqueue.PktQueue {
	return pktqueue.FromPtr(unsafe.Pointer(&server.c.rxQueue))
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
	return server.StopImpl(eal.NewStopFlag(unsafe.Pointer(&server.c.stop)))
}

// Close the server.
// The thread must be stopped before calling this.
func (server *Server) Close() error {
	server.GetRxQueue().Close()
	eal.Free(server.c)
	return nil
}
