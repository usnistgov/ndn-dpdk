package facemgmt

import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/iface"
)

type IdArg struct {
	Id iface.FaceId
}

type BasicInfo struct {
	Id      iface.FaceId
	Locator iface.LocatorWrapper
}

func newBasicInfo(face iface.IFace) (b BasicInfo) {
	b.Id = face.GetFaceId()
	b.Locator.Locator = face.GetLocator()
	return b
}

type FaceInfo struct {
	Id      iface.FaceId
	Locator iface.LocatorWrapper
	IsDown  bool

	// Basic counters.
	Counters iface.Counters

	// Extended counters.
	// This is *dpdk.EthStats for EthFace, and nil for other types.
	ExCounters interface{}

	// Latency for TX packets since arrival/generation (in nanos).
	Latency running_stat.Snapshot
}
