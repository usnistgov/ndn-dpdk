package iface

/*
#include "iface.h"
*/
import "C"
import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/ndn"
)

// Interface for a face.
type IFace interface {
	getPtr() *C.Face
	GetFaceId() FaceId
	GetNumaSocket() dpdk.NumaSocket
	Close() error
	EnableThreadSafeTx(queueCapacity int) error
	TxBurst(pkts []ndn.Packet)
	ReadCounters() Counters
	ReadLatency() running_stat.Snapshot
}

var gFaces [FACEID_MAX]IFace
var gFaceIds []FaceId

// Get face by FaceId.
func Get(faceId FaceId) IFace {
	return gFaces[faceId]
}

// Put constructed face (non-thread-safe).
func Put(face IFace) {
	faceId := face.GetFaceId()
	if faceId.GetKind() == FaceKind_None {
		panic("invalid FaceId")
	}
	if gFaces[faceId] != nil {
		panic("duplicate FaceId")
	}
	gFaceIds = append(gFaceIds, faceId)
	gFaces[faceId] = face
	C.gFaces[faceId] = face.getPtr()
}

// List FaceIds.
func ListFaceIds() []FaceId {
	return gFaceIds
}

// Close all faces (for unit tests).
func CloseAll() {
	for _, faceId := range gFaceIds {
		gFaces[faceId].Close()
		gFaces[faceId] = nil
	}
	gFaceIds = nil
}
