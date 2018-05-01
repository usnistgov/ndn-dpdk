package facemgmt

import (
	"errors"

	"ndn-dpdk/iface"
	"ndn-dpdk/iface/faceuri"
)

// Function to create a face.
var CreateFace func(remote, local *faceuri.FaceUri) (iface.FaceId, error)

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
	reply.LocalUri = face.GetLocalUri().String()
	reply.RemoteUri = face.GetRemoteUri().String()
	reply.IsDown = face.IsDown()
	reply.Counters = face.ReadCounters()
	reply.ExCounters = face.ReadExCounters()
	reply.Latency = face.ReadLatency()

	return nil
}

func (FaceMgmt) Create(args CreateArg, reply *IdArg) error {
	if CreateFace == nil {
		return errors.New("face creation is unavailable")
	}

	remote, e := faceuri.Parse(args.RemoteUri)
	if e != nil {
		return e
	}

	var local *faceuri.FaceUri
	if args.LocalUri != "" {
		if local, e = faceuri.Parse(args.LocalUri); e != nil {
			return e
		}
	}

	reply.Id, e = CreateFace(remote, local)
	return e
}

func (FaceMgmt) Destroy(args IdArg, reply *struct{}) error {
	face := iface.Get(args.Id)
	if face == nil {
		return errors.New("face not found")
	}

	return face.Close()
}
