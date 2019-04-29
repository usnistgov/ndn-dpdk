package facemgmt

import (
	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
	"ndn-dpdk/iface/faceuri"
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

type CreateArg struct {
	LocalUri  *faceuri.FaceUri
	RemoteUri *faceuri.FaceUri
}

func (a CreateArg) toIfaceCreateArg() (c createface.CreateArg) {
	c.Remote = a.RemoteUri
	c.Local = a.LocalUri
	return c
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
