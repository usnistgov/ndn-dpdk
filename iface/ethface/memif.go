package ethface

import (
	"errors"
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn/memiftransport"
)

const schemeMemif = "memif"

var memifVdevMap = make(map[string]*eal.VDev)

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
	name := "net_memif" + eal.AllocObjectID("ethface.Memif")
	key, args, e := loc.ToVDevArgs()
	if e != nil {
		return nil, fmt.Errorf("memif.Locator.ToVDevArgs %w", e)
	}
	if _, ok := memifVdevMap[key]; ok {
		return nil, errors.New("memif.Locator duplicate SocketName+ID with existing device")
	}

	vdev, e := eal.NewVDev(name, args, eal.NumaSocket{})
	if e != nil {
		return nil, fmt.Errorf("eal.NewVDev(%s,%s) %w", name, args, e)
	}
	memifVdevMap[key] = vdev
	dev := ethdev.Find(vdev.Name())

	pc := PortConfig{
		MTU:           loc.Dataroom,
		DisableSetMTU: true,
	}
	port, e := NewPort(dev, pc)
	if e != nil {
		vdev.Close()
		return nil, fmt.Errorf("NewPort %w", e)
	}
	port.closeVdev = func() {
		vdev := memifVdevMap[key]
		if vdev != nil && vdev.Close() == nil {
			delete(memifVdevMap, key)
		}
	}

	return New(port, loc)
}

func init() {
	iface.RegisterLocatorType(MemifLocator{}, schemeMemif)
}
