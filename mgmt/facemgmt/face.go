package facemgmt

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/createface"
)

type FaceMgmt struct{}

func (FaceMgmt) List(args struct{}, reply *[]BasicInfo) error {
	result := make([]BasicInfo, 0)
	for _, face := range iface.List() {
		result = append(result, makeBasicInfo(face))
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
	reply.IsDown = iface.IsDown(face.ID())
	reply.Counters = face.ReadCounters()
	reply.ExCounters = face.ReadExCounters()

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
	Id iface.ID
}

type BasicInfo struct {
	Id      iface.ID
	Locator iface.LocatorWrapper
}

func makeBasicInfo(face iface.Face) (b BasicInfo) {
	b.Id = face.ID()
	b.Locator.Locator = face.Locator()
	return b
}

type FaceInfo struct {
	BasicInfo
	IsDown bool

	// General counters.
	Counters iface.Counters

	// Extended counters.
	ExCounters interface{}
}
