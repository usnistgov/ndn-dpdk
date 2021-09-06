//go:build linux && amd64

package gqlmgmt

import (
	"context"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/usnistgov/ndn-dpdk/ndn/l3"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
	"go4.org/must"
)

// OpenFace invokes OpenMemif with default settings.
func (c *Client) OpenFace() (mgmt.Face, error) {
	return c.OpenMemif(memiftransport.Locator{})
}

const autoSocketPath = "/run/ndn"

var (
	autoSocketOnce sync.Once
	autoSocketName = ""
	autoMemifID    uint32
)

// OpenMemif creates a face connected to the current application using memif transport.
// If loc.SocketName is empty:
//  - loc.SocketName and loc.ID are automatically assigned
//  - loc.SocketOwner is set to current uid:gid
func (c *Client) OpenMemif(loc memiftransport.Locator) (mgmt.Face, error) {
	if loc.SocketName == "" {
		autoSocketOnce.Do(func() {
			autoSocketName = fmt.Sprintf("%s/memif-%d-%d.sock", autoSocketPath, os.Getpid(), time.Now().UnixNano())
			if e := os.MkdirAll(autoSocketPath, os.ModePerm); e != nil {
				panic(e)
			}
		})
		loc.SocketName = autoSocketName
		loc.ID = int(atomic.AddUint32(&autoMemifID, 1))
		loc.SocketOwner = &[2]int{os.Getuid(), os.Getgid()}
	}
	loc.ApplyDefaults(memiftransport.RoleClient)

	locR := loc.ReverseRole()
	locJ, e := locR.ToCreateFaceLocator()
	if e != nil {
		return nil, fmt.Errorf("loc.ToCreateFaceLocator: %w", e)
	}
	var faceJ faceJSON
	e = c.Do(context.TODO(), `
		mutation createFace($locator: JSON!) {
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
		routes:   make(map[string]string),
	}
	return f, f.openMemif(loc)
}

func (f *face) openMemif(loc memiftransport.Locator) error {
	tr, e := memiftransport.New(loc)
	if e != nil {
		must.Close(f)
		return fmt.Errorf("memiftransport.New: %w", e)
	}

	tr.OnStateChange(func(st l3.TransportState) {
		if st == l3.TransportClosed {
			must.Close(f)
		}
	})

	f.l3face, e = l3.NewFace(tr, l3.FaceConfig{})
	if e != nil {
		close(tr.Tx())
		return fmt.Errorf("l3.NewFace: %w", e)
	}
	return nil
}
