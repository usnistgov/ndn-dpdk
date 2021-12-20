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

func (loc Locator) EthCLocator() (c ethport.CLocator) {
	return
}

func (loc Locator) EthFaceConfig() (cfg ethport.FaceConfig) {
	return
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
		dev.Stop(ethdev.StopDetach)
		return nil, fmt.Errorf("NewPort %w", e)
	}

	return ethport.NewFace(port, loc)
}

func init() {
	iface.RegisterLocatorType(Locator{}, schemeMemif)
}
