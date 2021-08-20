//go:build !linux || !amd64

package gqlmgmt

import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/ndn/mgmt"
)

var errNoOpenFace = errors.New("OpenFace not supported on this platform")

// OpenFace returns an error on unsupported platform.
func (c *Client) OpenFace() (mgmt.Face, error) {
	return nil, errNoOpenFace
}
