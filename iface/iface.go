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
// Most functions are implemented by FaceBase type.
type IFace interface {
	getPtr() *C.Face

	// Get FaceId.
	GetFaceId() FaceId

	// Get a Locator describing face endpoints.
	// Lower layer implementation must provide this method.
	GetLocator() Locator

	// Get a FaceUri representing the local endpoint.
	// Lower layer implementation must provide this method.
	GetLocalUri() *faceuri.FaceUri

	// Get a FaceUri representing the remote endpoint.
	// Lower layer implementation must provide this method.
	GetRemoteUri() *faceuri.FaceUri

	// Get NUMA socket of this face's data structures.
	GetNumaSocket() dpdk.NumaSocket

	// Determine whether the face has been closed.
	IsClosed() bool

	// Close the face.
	// Lower layer implementation must provide this method.
	// It should return nil if FaceBase.IsClosed() returns true.
	// It should call FaceBase.BeforeClose and FaceBase.CloseFaceBase.
	Close() error

	// Determine whether the face is DOWN or UP.
	IsDown() bool

	// Get RxGroups that contain this face.
	ListRxGroups() []IRxGroup

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
