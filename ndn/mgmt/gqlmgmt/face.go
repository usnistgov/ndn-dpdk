package gqlmgmt

import (
	"fmt"
	"os"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
)

var lastMemifID = 0

// OpenFace invokes OpenMemif with default settings.
func (c *Client) OpenFace() (mgmt.Face, error) {
	return c.OpenMemif(memiftransport.Locator{})
}

// OpenMemif creates a face connected to the current application using memif transport.
// If loc.SocketName is empty, loc.SocketName and loc.ID will be assigned automatically.
func (c *Client) OpenMemif(loc memiftransport.Locator) (mgmt.Face, error) {
	if loc.SocketName == "" {
		loc.SocketName = fmt.Sprintf("/tmp/ndndpdk-memif-%d.sock", os.Getpid())
		lastMemifID++
		loc.ID = lastMemifID
	}
	loc.ApplyDefaults()

	locJ, e := loc.ToCreateFaceLocator()
	if e != nil {
		return nil, fmt.Errorf("loc.ToCreateFaceLocator: %w", e)
	}
	var faceJ faceJSON
	e = c.Do(`
		mutation createFace($locator: JSON) {
			createFace(locator: $locator) {
				id
			}
		}
	`, map[string]interface{}{
		"locator": locJ,
	}, "createFace", &faceJ)
	if e != nil {
		return nil, e
	}

	f := &face{
		faceJSON: faceJ,
		client:   c,
	}
	return f, f.open(loc)
}

type faceJSON struct {
	ID string `json:"id"`
}

type face struct {
	faceJSON
	client *Client
	l3face l3.Face
}

func (f *face) open(loc memiftransport.Locator) error {
	tr, e := memiftransport.New(loc)
	if e != nil {
		f.Close()
		return fmt.Errorf("memiftransport.New: %w", e)
	}

	tr.OnStateChange(func(st l3.TransportState) {
		if st == l3.TransportClosed {
			f.Close()
		}
	})

	f.l3face, e = l3.NewFace(tr)
	if e != nil {
		close(tr.Tx())
		return fmt.Errorf("l3.NewFace: %w", e)
	}
	return nil
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
