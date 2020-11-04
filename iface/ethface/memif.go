package ethface

import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

const schemeMemif = "memif"

// MemifLocator describes a memif face.
type MemifLocator struct {
	memiftransport.Locator
}

// Scheme returns "memif".
func (loc MemifLocator) Scheme() string {
	return schemeMemif
}

func (loc MemifLocator) conflictsWith(other ethLocator) bool {
	return true
}

func (loc MemifLocator) cLoc() (c cLocator) {
	copy(c.Local.Bytes[:], []uint8(memiftransport.AddressDPDK))
	copy(c.Remote.Bytes[:], []uint8(memiftransport.AddressApp))
	return
}

// CreateFace creates a memif face.
func (loc MemifLocator) CreateFace() (iface.Face, error) {
	name := "net_memif" + eal.AllocObjectID("ethface.Memif")
	args, e := loc.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("memif.Locator.ToVDevArgs %w", e)
	}

	vdev, e := eal.NewVDev(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("eal.NewVDev(%s,%s) %w", name, args, e)
	}
	dev := ethdev.Find(vdev.Name())

	var pc PortConfig
	pc.MTU = loc.Dataroom
	pc.NoSetMTU = true
	port, e := NewPort(dev, pc)
	if e != nil {
		vdev.Close()
		return nil, fmt.Errorf("NewPort %w", e)
	}
	port.vdev = vdev

	return New(port, loc)
}

func init() {
	iface.RegisterLocatorType(MemifLocator{}, schemeMemif)
}
