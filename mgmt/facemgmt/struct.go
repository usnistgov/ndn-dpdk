package facemgmt

import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/iface"
)

type IdArg struct {
	Id iface.FaceId
}

type CreateArg struct {
	LocalUri  string
	RemoteUri string
}

type FaceInfo struct {
	Id        iface.FaceId
	LocalUri  string
	RemoteUri string

	// Basic counters.
	Counters iface.Counters

	// Extended counters.
	// This is *dpdk.EthStats for EthFace, and nil for other types.
	ExCounters interface{}

	// Latency for TX packets since arrival/generation (in nanos).
	Latency running_stat.Snapshot
}
