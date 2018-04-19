package facemgmt

import (
	"errors"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
	"ndn-dpdk/iface/faceuri"
)

// Function to create a face.
var CreateFace func(u faceuri.FaceUri) (iface.FaceId, error)

type FaceMgmt struct{}

func (FaceMgmt) List(args struct{}, reply *[]iface.FaceId) error {
	list := make([]iface.FaceId, 0)
	for it := iface.IterFaces(); it.Valid(); it.Next() {
		list = append(list, it.Id)
	}
	*reply = list
	return nil
}

func (FaceMgmt) Get(args IdArg, reply *FaceInfo) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("face not found")
	}

	reply.Id = face.GetFaceId()
	reply.RemoteFaceUri = face.GetFaceUri().String()
	reply.Counters = face.ReadCounters()
	reply.Latency = face.ReadLatency()

	if reply.Id.GetKind() == iface.FaceKind_Eth {
		ethStats := face.(*ethface.EthFace).GetPort().GetStats()
		reply.EthStats = &ethStats
	}

	return nil
}

func (FaceMgmt) Create(args FaceUriArg, reply *IdArg) error {
	if CreateFace == nil {
		return errors.New("face creation is unavailable")
	}

	u, e := faceuri.Parse(args.RemoteFaceUri)
	if e != nil {
		return e
	}

	reply.Id, e = CreateFace(*u)
	return e
}
