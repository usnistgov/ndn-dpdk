package ethport

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"errors"

	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
)

type rxMemif struct{}

func (rxMemif) String() string {
	return "RxMemif"
}

func (impl *rxMemif) Init(port *Port) error {
	if port.devInfo.DriverName() != ethdev.DriverMemif {
		return errors.New("cannot use RxMemif on non-memif port")
	}
	return nil
}

func (impl *rxMemif) Start(face *Face) error {
	if e := face.port.startDev(1, false); e != nil {
		return e
	}
	C.EthFace_SetupRxMemif(face.priv, face.cLoc.ptr())
	rxf := &rxgFlow{
		face:  face,
		index: 0,
		queue: 0,
	}
	face.rxf = []*rxgFlow{rxf}
	iface.ActivateRxGroup(rxf)
	return nil
}

func (impl *rxMemif) Stop(face *Face) error {
	for _, rxf := range face.rxf {
		iface.DeactivateRxGroup(rxf)
	}
	face.rxf = nil
	return nil
}

func (impl *rxMemif) Close(port *Port) error {
	port.dev.Stop(ethdev.StopReset)
	return nil
}
