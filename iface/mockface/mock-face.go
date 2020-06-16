package mockface

/*
#include "../../csrc/iface/face.h"
uint16_t go_MockFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"io"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/emission"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/pktmbuf"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

const (
	evt_TxInterest = iota
	evt_TxData
	evt_TxNack
	evt_TxBadPkt
)

var (
	headerMempool   *pktmbuf.Pool
	nameMempool     *pktmbuf.Pool
	mempoolInitOnce sync.Once
)

type MockFace struct {
	iface.FaceBase

	emitter     *emission.Emitter
	txRecorders []io.Closer

	TxInterests []*ndni.Interest // sent Interest packets
	TxData      []*ndni.Data     // sent Data packets
	TxNacks     []*ndni.Nack     // sent Nack packets
	TxBadPkts   []ndni.Packet    // sent unparsable packets
}

func New() (face *MockFace) {
	mempoolInitOnce.Do(func() {
		headerMempool = ndni.HeaderMempool.MakePool(eal.NumaSocket{})
		nameMempool = ndni.NameMempool.MakePool(eal.NumaSocket{})
	})

	face = new(MockFace)

	face.emitter = emission.NewEmitter()
	face.EnableTxRecorders()

	if e := face.InitFaceBase(iface.AllocId(iface.FaceKind_Mock), 0, eal.NumaSocket{}); e != nil {
		panic(e)
	}
	iface.TheChanRxGroup.AddFace(face)

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.go_MockFace_TxBurst)

	if e := face.FinishInitFaceBase(256, 0, 0); e != nil {
		panic(e)
	}
	iface.Put(face)
	return face
}

func (face *MockFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

func (*MockFace) GetLocator() iface.Locator {
	return NewLocator()
}

func (face *MockFace) Close() error {
	if face.IsClosed() {
		return nil
	}
	face.BeforeClose()
	iface.TheChanRxGroup.RemoveFace(face)
	face.CloseFaceBase()
	return nil
}

func (face *MockFace) ListRxGroups() []iface.IRxGroup {
	return []iface.IRxGroup{iface.TheChanRxGroup}
}

// Cause the face to receive a packet.
// MockFace takes ownership of the underlying mbuf.
func (face *MockFace) Rx(l3pkt ndni.IL3Packet) {
	var lph ndni.LpHeader
	lph.LpL3 = *l3pkt.GetPacket().GetLpL3()

	pkt := l3pkt.GetPacket().AsMbuf()
	payloadL := pkt.Len()
	if pkt.GetHeadroom() <= ndni.PrependLpHeader_GetHeadroom() {
		hdrMbufs, e := headerMempool.Alloc(1)
		if e != nil {
			pkt.Close()
			return
		}
		hdr := hdrMbufs[0]
		hdr.SetHeadroom(ndni.PrependLpHeader_GetHeadroom())
		e = hdr.Chain(pkt)
		if e != nil {
			hdr.Close()
			pkt.Close()
			return
		}
		pkt = hdr
	} else {
		C.Packet_SetL2PktType((*C.Packet)(pkt.GetPtr()), C.L2PktType_None)
		C.Packet_SetL3PktType((*C.Packet)(pkt.GetPtr()), C.L3PktType_None)
	}

	// restore LpHeader because RxProc_Input will re-parse
	lph.Prepend(pkt, payloadL)

	pkt.SetPort(uint16(face.GetFaceId()))
	pkt.SetTimestamp(eal.TscNow())
	iface.TheChanRxGroup.Rx(pkt)
}

func (face *MockFace) OnTxInterest(cb func(interest *ndni.Interest)) io.Closer {
	return face.emitter.On(evt_TxInterest, cb)
}

func (face *MockFace) OnTxData(cb func(data *ndni.Data)) io.Closer {
	return face.emitter.On(evt_TxData, cb)
}

func (face *MockFace) OnTxNack(cb func(nack *ndni.Nack)) io.Closer {
	return face.emitter.On(evt_TxNack, cb)
}

func (face *MockFace) OnTxBadPkt(cb func(pkt ndni.Packet)) io.Closer {
	return face.emitter.On(evt_TxBadPkt, cb)
}

func (face *MockFace) EnableTxRecorders() {
	face.DisableTxRecorders()
	face.txRecorders = []io.Closer{
		face.OnTxInterest(func(interest *ndni.Interest) { face.TxInterests = append(face.TxInterests, interest) }),
		face.OnTxData(func(data *ndni.Data) { face.TxData = append(face.TxData, data) }),
		face.OnTxNack(func(nack *ndni.Nack) { face.TxNacks = append(face.TxNacks, nack) }),
		face.OnTxBadPkt(func(pkt ndni.Packet) { face.TxBadPkts = append(face.TxBadPkts, pkt) }),
	}
}

func (face *MockFace) DisableTxRecorders() {
	for _, closer := range face.txRecorders {
		closer.Close()
	}
	face.txRecorders = nil
}

func (face *MockFace) handleTx(pkt *ndni.Packet) {
	e := pkt.ParseL2()
	if e == nil {
		e = pkt.ParseL3(nameMempool)
	}
	if e != nil {
		face.emitter.EmitSync(evt_TxBadPkt, pkt)
		return
	}

	switch pkt.GetL3Type() {
	case ndni.L3PktType_Interest:
		face.emitter.EmitSync(evt_TxInterest, pkt.AsInterest())
	case ndni.L3PktType_Data:
		face.emitter.EmitSync(evt_TxData, pkt.AsData())
	case ndni.L3PktType_Nack:
		face.emitter.EmitSync(evt_TxNack, pkt.AsNack())
	}
}

//export go_MockFace_TxBurst
func go_MockFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.FaceId(faceC.id)).(*MockFace)
	for i := C.uint16_t(0); i < nPkts; i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		pkt := ndni.PacketFromPtr(unsafe.Pointer(*pktsEle))
		face.handleTx(pkt)
	}
	return C.uint16_t(nPkts)
}
