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

func (rxMemif) List(port *Port) (list []iface.RxGroup) {
	for _, face := range port.faces {
		list = append(list, face.rxf[0])
	}
	return
}

func (impl *rxMemif) Init(port *Port) error {
	if port.devInfo.Driver() != ethdev.DriverMemif {
		return errors.New("cannot use RxMemif on non-memif port")
	}
	return nil
}

func (impl *rxMemif) Start(face *Face) error {
	if e := face.port.startDev(1, false); e != nil {
		return e
	}

	cLoc := face.loc.EthCLocator()
	C.EthFace_SetupRxMemif(face.priv, cLoc.ptr())

	rxf := &rxgFlow{
		face:  face,
		index: 0,
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
	return nil
}
