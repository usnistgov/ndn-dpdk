package socketface

/*
#include "../face.h"
uint16_t go_SocketFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

// Configuration for creating SocketFace.
type Config struct {
	iface.Mempools
	RxMp      dpdk.PktmbufPool // mempool for received frames, dataroom must fit NDNLP frame
	TxqPkts   int              // before-TX queue capacity
	TxqFrames int              // after-TX queue capacity
}

// A face using socket as transport.
type SocketFace struct {
	iface.FaceBase
	logger logrus.FieldLogger
	conn   atomic.Value
	impl   iImpl

	closing   int32          // 1 if face is closing, need atomic access
	nRedials  int            // how many times face is redialed
	redialing int32          // 1 if face is redialing, need atomic access
	quitWg    sync.WaitGroup // wait until rxLoop and txLoop quits

	rxMp dpdk.PktmbufPool

	txQueue chan dpdk.Packet
}

// Create a SocketFace on a net.Conn.
func New(conn net.Conn, cfg Config) (face *SocketFace, e error) {
	face = new(SocketFace)
	network := conn.LocalAddr().Network()
	if impl, ok := implByNetwork[network]; ok {
		face.impl = impl
	} else {
		return nil, fmt.Errorf("unknown network %s", network)
	}

	if e := face.InitFaceBase(iface.AllocId(iface.FaceKind_Socket), 0, dpdk.NumaSocket{}); e != nil {
		return nil, e
	}

	face.logger = newLogger(face.GetFaceId())
	face.conn.Store(conn)
	face.rxMp = cfg.RxMp
	face.txQueue = make(chan dpdk.Packet, cfg.TxqFrames)

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.go_SocketFace_TxBurst)
	if e := face.FinishInitFaceBase(cfg.TxqPkts, 0, 0, cfg.Mempools); e != nil {
		return nil, e
	}

	iface.TheChanRxGroup.AddFace(face)
	face.quitWg.Add(2)
	go face.rxLoop()
	go face.txLoop()

	iface.Put(face)
	face.logger.Infof("new %s face %s->%s", conn.LocalAddr().Network(), conn.LocalAddr(), conn.RemoteAddr())
	return face, nil
}

func (face *SocketFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

func (face *SocketFace) GetConn() net.Conn {
	return face.conn.Load().(net.Conn)
}

func (face *SocketFace) GetLocator() iface.Locator {
	conn := face.GetConn()
	laddr, raddr := conn.LocalAddr(), conn.RemoteAddr()

	var loc Locator
	loc.Scheme = raddr.Network()
	loc.Remote = raddr.String()
	if laddr != nil {
		loc.Local = laddr.String()
	}
	return loc
}

func (face *SocketFace) Close() error {
	face.BeforeClose()
	atomic.StoreInt32(&face.closing, 1)
	close(face.txQueue)
	face.GetConn().Close() // ignore error
	face.quitWg.Wait()
	iface.TheChanRxGroup.RemoveFace(face)
	face.CloseFaceBase()
	return nil
}

func (face *SocketFace) ListRxGroups() []iface.IRxGroup {
	return []iface.IRxGroup{iface.TheChanRxGroup}
}

func (face *SocketFace) rxLoop() {
	face.impl.RxLoop(face)
	face.quitWg.Done()
}

// Report congestion when RxLoop is unable to send into rxQueue.
func (face *SocketFace) rxPkt(pkt dpdk.Packet) {
	pkt.SetPort(uint16(face.GetFaceId()))
	pkt.SetTimestamp(dpdk.TscNow())
	iface.TheChanRxGroup.Rx(pkt)
}

func (face *SocketFace) txLoop() {
	for {
		pkt, ok := <-face.txQueue
		if !ok {
			break
		}
		e := face.impl.Send(face, pkt)
		pkt.Close()
		if e != nil && face.handleError("TX", e) {
			break
		}
	}
	face.quitWg.Done()
}

// Handle socket error.
// Return whether RxLoop or TxLoop should terminate (i.e. face closed).
func (face *SocketFace) handleError(dir string, e error) bool {
	if atomic.LoadInt32(&face.closing) != 0 {
		return true
	}
	if netErr, ok := e.(net.Error); ok && netErr.Temporary() {
		face.logger.WithError(e).Errorf("%s socket error", dir)
		return false
	}
	face.logger.WithError(e).Errorf("%s socket failed", dir)

	if atomic.CompareAndSwapInt32(&face.redialing, 0, 1) {
		defer atomic.StoreInt32(&face.redialing, 0)
		for atomic.LoadInt32(&face.closing) == 0 {
			face.SetDown(true)
			time.Sleep(time.Second) // TODO exponential backoff
			face.nRedials++
			conn := face.GetConn()
			conn, e = face.impl.Redial(conn)
			if e == nil {
				face.logger.Infof("redialed %s->%s", conn.LocalAddr(), conn.RemoteAddr())
				face.conn.Store(conn)
				face.SetDown(false)
				break
			}
			face.logger.WithError(e).Errorf("redial failed")
		}
	} else { // another goroutine is redialing
		for atomic.LoadInt32(&face.redialing) != 0 {
			runtime.Gosched()
		}
	}
	return atomic.LoadInt32(&face.closing) != 0
}

//export go_SocketFace_TxBurst
func go_SocketFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.FaceId(faceC.id)).(*SocketFace)
	nQueued := C.uint16_t(0)
	for i := C.uint16_t(0); i < nPkts; i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		pkt := dpdk.MbufFromPtr(unsafe.Pointer(*pktsEle)).AsPacket()
		select {
		case face.txQueue <- pkt:
			nQueued++
		default:
			return nQueued
		}
	}
	return nQueued
}
