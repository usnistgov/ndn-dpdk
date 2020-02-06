package facemgmt

import (
	"errors"

	"ndn-dpdk/core/running_stat"
	"ndn-dpdk/iface"
	"ndn-dpdk/iface/createface"
)

type FaceMgmt struct{}

func (FaceMgmt) List(args struct{}, reply *[]BasicInfo) error {
	result := make([]BasicInfo, 0)
	for it := iface.IterFaces(); it.Valid(); it.Next() {
		result = append(result, makeBasicInfo(it.Face))
	}
	*reply = result
	return nil
}

func (FaceMgmt) Get(args IdArg, reply *FaceInfo) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("face not found")
	}

	reply.BasicInfo = makeBasicInfo(face)
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

	*reply = makeBasicInfo(face)
	return nil
}

func (FaceMgmt) Destroy(args IdArg, reply *struct{}) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("face not found")
	}

	return face.Close()
}

type IdArg struct {
	Id iface.FaceId
}

type BasicInfo struct {
	Id      iface.FaceId
	Locator iface.LocatorWrapper
}

func makeBasicInfo(face iface.IFace) (b BasicInfo) {
	b.Id = face.GetFaceId()
	b.Locator.Locator = face.GetLocator()
	return b
}

type FaceInfo struct {
	BasicInfo
	IsDown bool

	// Basic counters.
	Counters iface.Counters

	// Extended counters.
	ExCounters interface{}

	// Latency for TX packets since arrival/generation (in nanos).
	Latency running_stat.Snapshot
}
