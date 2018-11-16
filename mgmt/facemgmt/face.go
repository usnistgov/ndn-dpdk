package facemgmt

import (
	"errors"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
)

type FaceMgmt struct{}

func (FaceMgmt) List(args struct{}, reply *[]BasicInfo) error {
	result := make([]BasicInfo, 0)
	for it := iface.IterFaces(); it.Valid(); it.Next() {
		result = append(result, newBasicInfo(it.Face))
	}
	*reply = result
	return nil
}

func (FaceMgmt) Get(args IdArg, reply *FaceInfo) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("face not found")
	}

	reply.Id = face.GetFaceId()
	reply.LocalUri = face.GetLocalUri()
	reply.RemoteUri = face.GetRemoteUri()
	reply.IsDown = face.IsDown()
	reply.Counters = face.ReadCounters()
	reply.ExCounters = face.ReadExCounters()
	reply.Latency = face.ReadLatency()

	return nil
}

func (FaceMgmt) Create(args []CreateArg, reply *[]BasicInfo) (e error) {
	var list []createface.CreateArg
	for _, a := range args {
		list = append(list, a.toIfaceCreateArg())
	}

	faces, e := createface.Create(list...)
	if e != nil {
		return e
	}

	result := make([]BasicInfo, 0)
	for _, face := range faces {
		result = append(result, newBasicInfo(face))
	}
	*reply = result
	return nil
}

func (FaceMgmt) Destroy(args IdArg, reply *struct{}) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("face not found")
	}

	return face.Close()
}
