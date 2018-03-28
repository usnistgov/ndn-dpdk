package facemgmt

import (
	"errors"

	"ndn-dpdk/iface"
)

type FaceMgmt struct {
	ft IFaceTable
}

func New(ft IFaceTable) *FaceMgmt {
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
	return nil
}
