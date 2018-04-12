package facemgmt

import (
	"errors"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

type FaceMgmt struct{}

func (fm FaceMgmt) List(args struct{}, reply *[]iface.FaceId) error {
	*reply = iface.ListFaceIds()
	return nil
}

func (fm FaceMgmt) Get(args IdArg, reply *FaceInfo) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("Face not found.")
	}

	reply.Id = face.GetFaceId()
	reply.Counters = face.ReadCounters()
	reply.Latency = face.ReadLatency()

	if reply.Id.GetKind() == iface.FaceKind_Eth {
		ethStats := face.(*ethface.EthFace).GetPort().GetStats()
		reply.EthStats = &ethStats
	}

	return nil
}
