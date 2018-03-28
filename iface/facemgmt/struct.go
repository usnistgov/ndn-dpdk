package facemgmt

import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/iface"
)

type IdArg struct {
	Id iface.FaceId
}

type FaceInfo struct {
	Id       iface.FaceId
	Counters iface.Counters
	Latency  running_stat.Snapshot
}
