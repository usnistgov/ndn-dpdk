// Package memifface implements memif faces.
package memifface

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/iface/ethport"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

const schemeMemif = "memif"

// Locator describes a memif face.
type Locator struct {
	memiftransport.Locator
}

var _ ethport.Locator = Locator{}

// Scheme returns "memif".
func (Locator) Scheme() string {
	return schemeMemif
}

// EthCLocator implements ethport.Locator interface.
func (loc Locator) EthCLocator() (c ethport.CLocator) {
	return
}

// EthFaceConfig implements ethport.Locator interface.
func (loc Locator) EthFaceConfig() (cfg ethport.FaceConfig) {
	return ethport.FaceConfig{
		DisableTxMultiSegOffload: true,
	}
}

// CreateFace creates a memif face.
func (loc Locator) CreateFace() (iface.Face, error) {
	if e := loc.Locator.Validate(); e != nil {
		return nil, e
	}
	loc.Locator.ApplyDefaults(memiftransport.RoleServer)

	dev, e := ethdev.NewMemif(loc.Locator)
	if e != nil {
		return nil, e
	}

	port, e := ethport.New(ethport.Config{
		EthDev:    dev,
		MTU:       loc.Dataroom,
		AutoClose: true,
	})
	if e != nil {
		dev.Close()
		return nil, fmt.Errorf("NewPort %w", e)
	}

	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorScheme[Locator](schemeMemif)
}
