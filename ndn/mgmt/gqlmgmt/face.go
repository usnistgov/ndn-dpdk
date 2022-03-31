package gqlmgmt

import (
	"context"
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn"
	"github.com/usnistgov/ndn-dpdk/ndn/l3"
)

// Error conditions.
var (
	ErrFaceClosed = errors.New("face is closed")
)

type face struct {
	client *Client
	id     string
	l3face l3.Face
	routes map[string]string
}

func (f *face) ID() string {
	return f.id
}

func (f *face) Face() l3.Face {
	return f.l3face
}

func (f *face) Close() (e error) {
	if f.client == nil {
		return ErrFaceClosed
	}

	_, e = f.client.Delete(context.TODO(), f.ID())
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
	e := f.client.Do(context.TODO(), `
		mutation insertFibEntry($name: Name!, $nexthops: [ID!]!) {
			insertFibEntry(name: $name, nexthops: $nexthops) {
				id
			}
		}
	`, map[string]any{
		"name":     name.String(),
		"nexthops": []string{f.ID()},
	}, "insertFibEntry", &fibEntryJ)
	if e == nil {
		f.routes[string(nameV)] = fibEntryJ.ID
	}
	return e
}

func (f *face) Withdraw(name ndn.Name) (e error) {
	if f.client == nil {
		return ErrFaceClosed
	}

	nameV, _ := name.MarshalBinary()
	id, ok := f.routes[string(nameV)]
	if !ok {
		return nil
	}

	_, e = f.client.Delete(context.TODO(), id)
	if e == nil {
		delete(f.routes, string(nameV))
	}
	return e
}
