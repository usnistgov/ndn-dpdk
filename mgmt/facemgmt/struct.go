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
	Id        iface.FaceId
	LocalUri  *faceuri.FaceUri
	RemoteUri *faceuri.FaceUri
}

func newBasicInfo(face iface.IFace) (b BasicInfo) {
	b.Id = face.GetFaceId()
	b.LocalUri = face.GetLocalUri()
	b.RemoteUri = face.GetRemoteUri()
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
