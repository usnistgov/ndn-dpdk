package mockface

/*
#include "../face.h"
uint16_t go_MockFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"io"
	"unsafe"

	"ndn-dpdk/core/emission"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Face mempools.
// These must be assigned before calling New().
var FaceMempools iface.Mempools

const (
	evt_TxInterest = iota
	evt_TxData
	evt_TxNack
	evt_TxBadPkt
)

type MockFace struct {
	iface.FaceBase

	emitter     *emission.Emitter
	txRecorders []io.Closer

	TxInterests []*ndn.Interest // sent Interest packets
	TxData      []*ndn.Data     // sent Data packets
	TxNacks     []*ndn.Nack     // sent Nack packets
	TxBadPkts   []ndn.Packet    // sent unparsable packets
}

func New() (face *MockFace) {
	face = new(MockFace)

	face.emitter = emission.NewEmitter()
	face.EnableTxRecorders()

	if e := face.InitFaceBase(iface.AllocId(iface.FaceKind_Mock), 0, dpdk.NumaSocket{}); e != nil {
		panic(e)
	}
	iface.TheChanRxGroup.AddFace(face)

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.go_MockFace_TxBurst)

	if e := face.FinishInitFaceBase(256, 0, 0, FaceMempools); e != nil {
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
func (face *MockFace) Rx(l3pkt ndn.IL3Packet) {
	var lph ndn.LpHeader
	lph.LpL3 = *l3pkt.GetPacket().GetLpL3()

	pkt := l3pkt.GetPacket().AsDpdkPacket()
	payloadL := pkt.Len()
	if pkt.GetFirstSegment().GetHeadroom() <= ndn.PrependLpHeader_GetHeadroom() {
		hdrMbuf, e := FaceMempools.HeaderMp.Alloc()
		if e != nil {
			pkt.Close()
			return
		}
		hdr := hdrMbuf.AsPacket()
		hdr.GetFirstSegment().SetHeadroom(ndn.PrependLpHeader_GetHeadroom())
		e = hdr.AppendPacket(pkt)
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
	pkt.SetTimestamp(dpdk.TscNow())
	iface.TheChanRxGroup.Rx(pkt)
}

func (face *MockFace) OnTxInterest(cb func(interest *ndn.Interest)) io.Closer {
	return face.emitter.On(evt_TxInterest, cb)
}

func (face *MockFace) OnTxData(cb func(data *ndn.Data)) io.Closer {
	return face.emitter.On(evt_TxData, cb)
}

func (face *MockFace) OnTxNack(cb func(nack *ndn.Nack)) io.Closer {
	return face.emitter.On(evt_TxNack, cb)
}

func (face *MockFace) OnTxBadPkt(cb func(pkt ndn.Packet)) io.Closer {
	return face.emitter.On(evt_TxBadPkt, cb)
}

func (face *MockFace) EnableTxRecorders() {
	face.DisableTxRecorders()
	face.txRecorders = []io.Closer{
		face.OnTxInterest(func(interest *ndn.Interest) { face.TxInterests = append(face.TxInterests, interest) }),
		face.OnTxData(func(data *ndn.Data) { face.TxData = append(face.TxData, data) }),
		face.OnTxNack(func(nack *ndn.Nack) { face.TxNacks = append(face.TxNacks, nack) }),
		face.OnTxBadPkt(func(pkt ndn.Packet) { face.TxBadPkts = append(face.TxBadPkts, pkt) }),
	}
}

func (face *MockFace) DisableTxRecorders() {
	for _, closer := range face.txRecorders {
		closer.Close()
	}
	face.txRecorders = nil
}

func (face *MockFace) handleTx(pkt ndn.Packet) {
	e := pkt.ParseL2()
	if e == nil {
		e = pkt.ParseL3(FaceMempools.NameMp)
	}
	if e != nil {
		face.emitter.EmitSync(evt_TxBadPkt, pkt)
		return
	}

	switch pkt.GetL3Type() {
	case ndn.L3PktType_Interest:
		face.emitter.EmitSync(evt_TxInterest, pkt.AsInterest())
	case ndn.L3PktType_Data:
		face.emitter.EmitSync(evt_TxData, pkt.AsData())
	case ndn.L3PktType_Nack:
		face.emitter.EmitSync(evt_TxNack, pkt.AsNack())
	}
}

//export go_MockFace_TxBurst
func go_MockFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.FaceId(faceC.id)).(*MockFace)
	for i := C.uint16_t(0); i < nPkts; i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		pkt := ndn.PacketFromPtr(unsafe.Pointer(*pktsEle))
		face.handleTx(pkt)
	}
	return C.uint16_t(nPkts)
}
