package iface

/*
#include "face.h"
*/
import "C"
import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface/faceuri"
	"ndn-dpdk/ndn"
)

// Interface for a face.
// Most functions are implemented by BaseFace type.
type IFace interface {
	getPtr() *C.Face

	// Get FaceId.
	GetFaceId() FaceId

	// Get a FaceUri representing the local endpoint.
	// Lower layer implementation must provide this method.
	GetLocalUri() *faceuri.FaceUri

	// Get a FaceUri representing the remote endpoint.
	// Lower layer implementation must provide this method.
	GetRemoteUri() *faceuri.FaceUri

	// Get NUMA socket of this face's data structures.
	GetNumaSocket() dpdk.NumaSocket

	// Close the face.
	// Lower layer implementation must provide this method, and call BaseFace.CloseBaseFace.
	Close() error

	// Enable thread-safety on C.Face_TxBurst function.
	EnableThreadSafeTx(queueCapacity int) error

	// Transmit a burst of L3 packets.
	TxBurst(pkts []ndn.Packet)

	// Read basic face counters.
	ReadCounters() Counters

	// Read extended counters.
	// Lower layer implementation may override this method.
	ReadExCounters() interface{}

	// Read L3 latency statistics (in nanoseconds).
	ReadLatency() running_stat.Snapshot
}

var gFaces [int(FACEID_MAX) + 1]IFace

// Get face by FaceId.
func Get(faceId FaceId) IFace {
	return gFaces[faceId]
}

// Put face (non-thread-safe).
// This should be called by face subtype constructor.
func Put(face IFace) {
	faceId := face.GetFaceId()
	if faceId.GetKind() == FaceKind_None {
		panic("invalid FaceId")
	}
	if gFaces[faceId] != nil {
		panic("duplicate FaceId")
	}
	gFaces[faceId] = face
}

// Iterator over faces.
//
// Usage:
// for it := iface.IterFaces(); it.Valid(); it.Next() {
//   // use it.Id and it.Face
// }
type FaceIterator struct {
	Id   FaceId
	Face IFace
}

func IterFaces() *FaceIterator {
	var it FaceIterator
	it.Id = FACEID_INVALID
	it.Next()
	return &it
}

func (it *FaceIterator) Valid() bool {
	return it.Id <= FACEID_MAX
}

func (it *FaceIterator) Next() {
	for it.Id++; it.Id <= FACEID_MAX; it.Id++ {
		it.Face = gFaces[it.Id]
		if it.Face != nil {
			return
		}
	}
	it.Face = nil
}

// Close all faces.
func CloseAll() {
	for it := IterFaces(); it.Valid(); it.Next() {
		it.Face.Close()
	}
}
