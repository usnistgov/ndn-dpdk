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
	reply.Locator.Locator = face.GetLocator()
	reply.IsDown = face.IsDown()
	reply.Counters = face.ReadCounters()
	reply.ExCounters = face.ReadExCounters()
	reply.Latency = face.ReadLatency()

	return nil
}

func (FaceMgmt) Create(args iface.LocatorWrapper, reply *BasicInfo) (e error) {
	face, e := createface.Create(args.Locator)
	if e != nil {
		return e
	}

	*reply = newBasicInfo(face)
	return nil
}

func (FaceMgmt) Destroy(args IdArg, reply *struct{}) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("face not found")
	}

	return face.Close()
}
