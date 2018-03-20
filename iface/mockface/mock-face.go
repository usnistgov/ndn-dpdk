package mockface

/*
#include "mock-face.h"
*/
import "C"
import (
	"math/rand"
	"unsafe"

	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
	"ndn-dpdk/ndn"
)

// Face mempools.
// These must be assigned before calling New().
var FaceMempools iface.Mempools

const (
	minId = 0x0001
	maxId = 0x0FFF
)

var faceById [maxId - minId + 1]*MockFace

func getById(id int) *MockFace {
	return faceById[id-minId]
}

// Retrieve MockFace by FaceId.
func Get(id iface.FaceId) *MockFace {
	if id.GetKind() != iface.FaceKind_Mock {
		return nil
	}
	return getById(int(id))
}

func setById(id int, face *MockFace) {
	faceById[id-minId] = face
}

type MockFace struct {
	iface.Face
	isClosed bool

	TxInterests []*ndn.Interest // sent Interest packets
	TxData      []*ndn.Data     // sent Data packets
	TxNacks     []*ndn.Nack     // sent Nack packets
	TxBadPkts   []ndn.Packet    // send unparsable packets
}

func New() (face *MockFace) {
	id := 0
	for {
		id = minId + rand.Intn(maxId-minId+1)
		if getById(id) == nil {
			break
		}
	}

	face = new(MockFace)
	face.AllocCFace(C.sizeof_MockFace, dpdk.NUMA_SOCKET_ANY)

	C.MockFace_Init(face.getPtr(), C.FaceId(id),
		(*C.FaceMempools)(FaceMempools.GetPtr()))
	setById(id, face)

	return face
}

func (face *MockFace) getPtr() *C.MockFace {
	return (*C.MockFace)(face.GetPtr())
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

func getByCFace(faceC *C.Face) *MockFace {
	face := getById(int(faceC.id))
	if face == nil {
		panic("MockFace not found")
	}
	return face
}

//export go_MockFace_TxBurst
func go_MockFace_TxBurst(faceC *C.Face, pkts **C.struct_rte_mbuf, nPkts C.uint16_t) C.uint16_t {
	face := getByCFace(faceC)
	for i := C.uint16_t(0); i < nPkts; i++ {
		pktsEle := (**C.struct_rte_mbuf)(unsafe.Pointer(uintptr(unsafe.Pointer(pkts)) +
			uintptr(i)*unsafe.Sizeof(*pkts)))
		pkt := ndn.PacketFromPtr(unsafe.Pointer(*pktsEle))
		face.recordTx(pkt)
	}
	return C.uint16_t(nPkts)
}

//export go_MockFace_Close
func go_MockFace_Close(faceC *C.Face) C.bool {
	face := getByCFace(faceC)
	face.isClosed = true
	return C.bool(true)
}
