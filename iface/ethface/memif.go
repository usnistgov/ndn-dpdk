package ethface

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"errors"
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
	return
}

func (loc MemifLocator) faceConfig() FaceConfig {
	return FaceConfig{}
}

// CreateFace creates a memif face.
func (loc MemifLocator) CreateFace() (iface.Face, error) {
	if e := loc.Locator.Validate(); e != nil {
		return nil, e
	}
	loc.Locator.ApplyDefaults(memiftransport.RoleServer)

	dev, e := ethvdev.NewMemif(loc.Locator)
	if e != nil {
		return nil, e
	}

	pc := PortConfig{
		MTU:           loc.Dataroom,
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

type rxMemifImpl struct {
	port *Port
}

func (rxMemifImpl) String() string {
	return "RxMemif"
}

func (impl *rxMemifImpl) Init(port *Port) error {
	if port.dev.DevInfo().DriverName() != "net_memif" {
		return errors.New("cannot use RxMemif on non-memif port")
	}
	impl.port = port
	return nil
}

func (impl *rxMemifImpl) Start(face *ethFace) error {
	if e := startDev(face.port, 1, false); e != nil {
		return e
	}
	cLoc := face.loc.cLoc()
	C.EthFace_SetupRxMemif(face.priv, cLoc.ptr())
	rxf := &rxFlow{
		face:  face,
		index: 0,
		queue: 0,
	}
	face.rxf = []*rxFlow{rxf}
	iface.ActivateRxGroup(rxf)
	return nil
}

func (impl *rxMemifImpl) Stop(face *ethFace) error {
	for _, rxf := range face.rxf {
		iface.DeactivateRxGroup(rxf)
	}
	face.rxf = nil
	return nil
}

func (impl *rxMemifImpl) Close() error {
	if impl.port != nil {
		impl.port.dev.Stop(ethdev.StopReset)
	}
	return nil
}
