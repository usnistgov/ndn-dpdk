package socketface

/*
#include "../../csrc/socketface/face.h"
extern uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
STATIC_ASSERT_FUNC_TYPE(Face_TxBurstFunc, go_SocketFace_TxBurst);
*/
import "C"
import (
	"bytes"
	"context"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"
)

type txImpl struct {
	describe string
	txBurst  unsafe.Pointer
	start    func(face *socketFace)
}

func (impl *txImpl) String() string {
	return impl.describe
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.ID(faceC.id)).(*socketFace)
	vec := pktmbuf.VectorFromPtr(unsafe.Pointer(pkts), int(nPkts))
	defer vec.Close()
	for _, pkt := range vec {
		segs := pkt.SegmentBytes()
		if len(segs) == 1 {
			face.transport.Write(segs[0])
		} else {
			face.transport.Write(bytes.Join(segs, nil))
		}
	}
	return nPkts
}

var txConnImpl = txImpl{
	describe: "TxConn",
	txBurst:  C.go_SocketFace_TxBurst,
}

func (face *socketFace) acquireTxFd() {
	e := face.rawControl(func(ctx context.Context, fd int) error {
		logEntry := face.logger.With(zap.Int("fd", fd))
		logEntry.Debug("file descriptor acquired for socket TX")
		defer logEntry.Debug("file descriptor released for socket TX")

		face.priv.fd = C.int(fd)
		<-ctx.Done()
		face.priv.fd = -1
		return nil
	})
	if e != nil {
		face.logger.Error("cannot acquire file descriptor for socket TX; outgoing packets will be dropped", zap.Error(e))
	}
}

var txSyscallImpl = txImpl{
	describe: "TxSyscall",
	txBurst:  C.SocketFace_DgramTxBurst,
	start: func(face *socketFace) {
		go face.acquireTxFd()
	},
}
