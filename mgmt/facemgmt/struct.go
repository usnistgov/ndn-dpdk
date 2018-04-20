package facemgmt

import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type IdArg struct {
	Id iface.FaceId
}

type FaceUriArg struct {
	RemoteFaceUri string
}

type FaceInfo struct {
	Id        iface.FaceId
	LocalUri  string
	RemoteUri string

	// Basic counters.
	Counters iface.Counters

	// DPDK EthDev stats.
	EthStats *dpdk.EthStats

	// Latency for TX packets since arrival/generation (in nanos).
	Latency running_stat.Snapshot
}
