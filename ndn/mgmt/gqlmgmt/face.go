package gqlmgmt

import (
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

type faceJSON struct {
	ID string `json:"id"`
}

type face struct {
	faceJSON
	client *Client
	l3face l3.Face
}

func (f *face) ID() string {
	return f.faceJSON.ID
}

func (f *face) Face() l3.Face {
	return f.l3face
}

func (f *face) Close() error {
	if f.client == nil { // already closed
		return nil
	}

	e := f.client.Do(`
		mutation delete($id: ID!) {
			delete(id: $id)
		}
	`, f.faceJSON, "", nil)
	f.client = nil
	return e
}
