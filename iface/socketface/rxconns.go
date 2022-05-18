package socketface

/*
#include "../../csrc/socketface/rxconns.h"
*/
import "C"
import (
	"time"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/dpdk/ringbuffer"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
)

type rxConns struct {
	rxFaceList
	socket eal.NumaSocket
	ring   *ringbuffer.Ring
	mp     *pktmbuf.Pool
	c      *C.SocketRxConns
}

var _ iface.RxGroup = (*rxConns)(nil)

func (rxc *rxConns) NumaSocket() eal.NumaSocket {
	return rxc.socket
}

func (rxc *rxConns) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(rxc.c), "SocketRxConns"
}

func (rxc *rxConns) close() {
	iface.DeactivateRxGroup(rxc)
	eal.Free(rxc.c)
	rxc.ring.Close()
	rxc.c, rxc.ring = nil, nil
	logger.Info("RxConns closed")
}

func (rxc *rxConns) run(face *socketFace) error {
	defer rxc.faceListPut(face)()
	face.logger.Debug("face is using RxConns")
	id, ctx := face.ID(), face.transport.Context()
	for ctx.Err() == nil {
		vec, e := rxc.mp.Alloc(1)
		if e != nil { // alloc error, try again later
			time.Sleep(time.Millisecond)
			continue
		}
		pkt := vec[0]
		pkt.SetHeadroom(0)

		for {
			n, e := pkt.ReadFrom(face.transport)
			if e != nil {
				vec.Close()
				return e
			}
			if n > 0 {
				break
			}
		}

		pkt.SetPort(uint16(id))
		pkt.SetTimestamp(eal.TscNow())
		if ringbuffer.Enqueue(rxc.ring, vec) == 0 {
			pkt.Close()
		}
	}
	return nil
}

func newRxConns(ringCapacity int, socket eal.NumaSocket) (rxc *rxConns, e error) {
	rxc = &rxConns{
		socket: socket,
	}
	if rxc.ring, e = ringbuffer.New(ringCapacity, rxc.socket, ringbuffer.ProducerMulti, ringbuffer.ConsumerSingle); e != nil {
		return nil, e
	}
	rxc.mp = ndni.PacketMempool.Get(rxc.socket)

	rxc.c = eal.Zmalloc[C.SocketRxConns]("SocketRxConns", C.sizeof_SocketRxConns, rxc.socket)
	rxc.c.base.rxBurst = C.RxGroup_RxBurstFunc(C.SocketRxConns_RxBurst)
	rxc.c.ring = (*C.struct_rte_ring)(rxc.ring.Ptr())

	logger.Info("RxConns created",
		zap.Int("ring-capacity", rxc.ring.Capacity()),
		rxc.socket.ZapField("socket"),
	)
	iface.ActivateRxGroup(rxc)
	return rxc, nil
}

var rxConnsImpl = rxImpl{
	describe: "RxConns",
	nilValue: (*rxConns)(nil),
	create: func() (rxGroup, error) {
		return newRxConns(gCfg.RxConns.RingCapacity, gCfg.numaSocket())
	},
}
