package mockface

/*
#include "../face.h"
uint16_t go_MockFace_TxBurst(Face* faceC, struct rte_mbuf** pkts, uint16_t nPkts);
*/
import "C"
import (
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/ndn"
)

// Face mempools.
// These must be assigned before calling New().
var FaceMempools iface.Mempools

type MockFace struct {
	iface.BaseFace
	isClosed bool

	TxInterests []*ndn.Interest // sent Interest packets
	TxData      []*ndn.Data     // sent Data packets
	TxNacks     []*ndn.Nack     // sent Nack packets
	TxBadPkts   []ndn.Packet    // sent unparsable packets
}

func New() *MockFace {
	var face MockFace
	face.InitBaseFace(iface.AllocId(iface.FaceKind_Mock), 0, dpdk.NUMA_SOCKET_ANY)

	faceC := face.getPtr()
	faceC.txBurstOp = (C.FaceImpl_TxBurst)(C.go_MockFace_TxBurst)
	C.FaceImpl_Init(faceC, 0, 0, (*C.FaceMempools)(FaceMempools.GetPtr()))
	iface.Put(&face)
	return &face
}

func (face *MockFace) getPtr() *C.Face {
	return (*C.Face)(face.GetPtr())
}

func (*MockFace) GetLocalUri() *faceuri.FaceUri {
	return faceuri.MustParse("mock:")
}

func (*MockFace) GetRemoteUri() *faceuri.FaceUri {
	return faceuri.MustParse("mock:")
}

func (face *MockFace) Close() error {
	face.BeforeClose()
	face.isClosed = true
	face.CloseBaseFace()
	return nil
}

func (face *MockFace) IsClosed() bool {
	return face.isClosed
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

	pkt.SetTimestamp(dpdk.TscNow())
	rxQueue <- rxPacket{face, pkt}
}

func (face *MockFace) recordTx(pkt ndn.Packet) {
	e := pkt.ParseL2()
	if e == nil {
		e = pkt.ParseL3(FaceMempools.NameMp)
	}
	if e != nil {
		face.TxBadPkts = append(face.TxBadPkts, pkt)
		return
	}

	switch pkt.GetL3Type() {
	case ndn.L3PktType_Interest:
		face.TxInterests = append(face.TxInterests, pkt.AsInterest())
	case ndn.L3PktType_Data:
		face.TxData = append(face.TxData, pkt.AsData())
	case ndn.L3PktType_Nack:
		face.TxNacks = append(face.TxNacks, pkt.AsNack())
	}
}

//export go_MockFace_TxBurst
func go_MockFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := iface.Get(iface.FaceId(faceC.id)).(*MockFace)
	for i := C.uint16_t(0); i < nPkts; i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		pkt := ndn.PacketFromPtr(unsafe.Pointer(*pktsEle))
		face.recordTx(pkt)
	}
	return C.uint16_t(nPkts)
}
