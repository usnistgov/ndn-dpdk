package ethface

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethvdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

const schemeMemif = "memif"

// MemifLocator describes a memif face.
type MemifLocator struct {
	memiftransport.Locator
}

// Scheme returns "memif".
func (MemifLocator) Scheme() string {
	return schemeMemif
}

func (loc MemifLocator) cLoc() (c cLocator) {
	copy(c.Local.Bytes[:], []uint8(memiftransport.AddressDPDK))
	copy(c.Remote.Bytes[:], []uint8(memiftransport.AddressApp))
	return
}

func (loc MemifLocator) faceConfig() FaceConfig {
	return FaceConfig{}
}

// CreateFace creates a memif face.
func (loc MemifLocator) CreateFace() (iface.Face, error) {
	dev, e := ethvdev.NewMemif(loc.Locator)
	if e != nil {
		return nil, e
	}

	pc := PortConfig{
		MTU:           loc.Dataroom - 14, // Ethernet header is not part of MTU
		DisableSetMTU: true,
	}
	port, e := NewPort(dev, pc)
	if e != nil {
		dev.Stop(ethdev.StopDetach)
		return nil, fmt.Errorf("NewPort %w", e)
	}

	return New(port, loc)
}

func init() {
	iface.RegisterLocatorType(MemifLocator{}, schemeMemif)
}
