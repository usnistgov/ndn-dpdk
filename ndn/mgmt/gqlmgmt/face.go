package gqlmgmt

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Error conditions.
var (
	ErrFaceClosed = errors.New("face is closed")
)

type faceJSON struct {
	ID string `json:"id"`
}

type face struct {
	faceJSON
	client *Client
	l3face l3.Face
	routes map[string]string
}

func (f *face) ID() string {
	return f.faceJSON.ID
}

func (f *face) Face() l3.Face {
	return f.l3face
}

func (f *face) Close() error {
	if f.client == nil {
		return ErrFaceClosed
	}

	e := f.client.delete(f.ID())
	f.client = nil
	return e
}

func (f *face) Advertise(name ndn.Name) error {
	if f.client == nil {
		return ErrFaceClosed
	}

	nameV, _ := name.MarshalBinary()
	if _, ok := f.routes[string(nameV)]; ok {
		return nil
	}

	var fibEntryJ struct {
		ID string `json:"id"`
	}
	e := f.client.Do(`
		mutation insertFibEntry($name: Name!, $nexthops: [ID!]!, $strategy: ID) {
			insertFibEntry(name: $name, nexthops: $nexthops, strategy: $strategy) {
				id
			}
		}
	`, map[string]interface{}{
		"name":     name.String(),
		"nexthops": []string{f.ID()},
	}, "insertFibEntry", &fibEntryJ)
	if e == nil {
		f.routes[string(nameV)] = fibEntryJ.ID
	}
	return e
}

func (f *face) Withdraw(name ndn.Name) error {
	if f.client == nil {
		return ErrFaceClosed
	}

	nameV, _ := name.MarshalBinary()
	id, ok := f.routes[string(nameV)]
	if !ok {
		return nil
	}

	e := f.client.delete(id)
	if e == nil {
		delete(f.routes, string(nameV))
	}
	return e
}
