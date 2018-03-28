package facemgmt

import (
	"errors"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/ethface"
)

type FaceMgmt struct {
	ft iface.FaceTable
}

func New(ft iface.FaceTable) *FaceMgmt {
	return &FaceMgmt{ft}
}

func (fm *FaceMgmt) List(args struct{}, reply *[]iface.FaceId) error {
	faces := fm.ft.ListFaces()
	*reply = make([]iface.FaceId, len(faces))
	for i, face := range faces {
		(*reply)[i] = face.GetFaceId()
	}
	return nil
}

func (fm *FaceMgmt) Get(args IdArg, reply *FaceInfo) error {
	face := fm.ft.GetFace(args.Id)
	if !face.IsValid() {
		return errors.New("Face not found.")
	}

	reply.Id = face.GetFaceId()
	reply.Counters = face.ReadCounters()
	reply.Latency = face.ReadLatency()

	if face.GetFaceId().GetKind() == iface.FaceKind_EthDev {
		ethStats := ethface.EthFace{face}.GetPort().GetStats()
		reply.EthStats = &ethStats
	}

	return nil
}
