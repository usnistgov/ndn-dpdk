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
