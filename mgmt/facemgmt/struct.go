package facemgmt

import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
)

type IdArg struct {
	Id iface.FaceId
}

type CreateArg struct {
	LocalUri  *faceuri.FaceUri
	RemoteUri *faceuri.FaceUri
}

type FaceInfo struct {
	Id        iface.FaceId
	LocalUri  *faceuri.FaceUri
	RemoteUri *faceuri.FaceUri
	IsDown    bool

	// Basic counters.
	Counters iface.Counters

	// Extended counters.
	// This is *dpdk.EthStats for EthFace, and nil for other types.
	ExCounters interface{}

	// Latency for TX packets since arrival/generation (in nanos).
	Latency running_stat.Snapshot
}
