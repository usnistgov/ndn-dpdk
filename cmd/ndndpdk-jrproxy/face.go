package main

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
)

var (
	faceLock   sync.Mutex
	faceNidGid = map[int]string{}
)

var errNoFace = errors.New("face not found")

type Face struct{}

func (Face) List(args struct{}, reply *[]*FaceBasicInfo) error {
	e := client.Do(context.TODO(), `
		{
			faces {
				gID: id
				Id: nid
				Locator: locator
			}
		}
	`, nil, "faces", reply)
	if e != nil {
		return e
	}

	faceLock.Lock()
	defer faceLock.Unlock()
	for _, face := range *reply {
		faceNidGid[face.Nid] = face.Gid
		face.Gid = ""
	}
	return nil
}

func (Face) Get(args FaceIdArg, reply *FaceInfo) error {
	faceLock.Lock()
	defer faceLock.Unlock()
	gID := faceNidGid[args.Nid]
	if gID == "" {
		return errNoFace
	}

	e := client.Do(context.TODO(), `
		query getFace($id: ID!) {
			node(id: $id) {
				... on Face {
					Id: nid
					locator
					ethDev {
						isDown
					}
				}
			}
		}
	`, map[string]any{
		"id": gID,
	}, "node", reply)
	if e != nil {
		return e
	}

	if reply.EthDev != nil {
		reply.IsDown = reply.EthDev.IsDown
		reply.EthDev = nil
	}
	return nil
}

func (Face) Create(args any, reply *FaceBasicInfo) error {
	e := client.Do(context.TODO(), `
		mutation createFace($locator: JSON!) {
			createFace(locator: $locator) {
				gID: id
				Id: nid
				Locator: locator
			}
		}
	`, map[string]any{
		"locator": args,
	}, "createFace", reply)
	if e != nil {
		return e
	}

	faceLock.Lock()
	defer faceLock.Unlock()
	faceNidGid[reply.Nid] = reply.Gid
	reply.Gid = ""
	return nil
}

func (Face) Destroy(args FaceIdArg, reply *struct{}) error {
	faceLock.Lock()
	defer faceLock.Unlock()
	gID := faceNidGid[args.Nid]
	if gID == "" {
		return nil
	}

	e := client.Do(context.TODO(), `
		mutation delete($id: ID!) {
			delete(id: $id)
		}
	`, map[string]any{
		"id": gID,
	}, "", nil)
	if e != nil {
		return e
	}

	delete(faceNidGid, args.Nid)
	return nil
}

type FaceIdArg struct {
	Nid int `json:"Id"`
}

type FaceBasicInfo struct {
	Gid     string `json:"gID,omitempty"`
	Nid     int    `json:"Id"`
	Locator json.RawMessage
}

type FaceInfo struct {
	FaceBasicInfo
	IsDown bool
	EthDev *struct {
		IsDown bool
	} `json:",omitempty"`
}
