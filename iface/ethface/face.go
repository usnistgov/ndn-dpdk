// Package ethface implements Ethernet faces using DPDK Ethernet devices.
package ethface

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"fmt"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/iface"
)

// LocatorConflictError indicates that the locator of a new face conflicts with an existing face.
type LocatorConflictError struct {
	a, b ethLocator
}

func (e LocatorConflictError) Error() string {
	return fmt.Sprintf("locator %s conflicts with %s", iface.LocatorString(e.a), iface.LocatorString(e.b))
}

type ethLocator interface {
	iface.Locator

	// conflictsWith determines whether this and other locator can coexist on the same port.
	conflictsWith(other ethLocator) bool

	// cLoc converts to C.EthLocator.
	cLoc() cLocator
}

func (loc *cLocator) ptr() *C.EthLocator {
	return (*C.EthLocator)(unsafe.Pointer(loc))
}

type ethFace struct {
	iface.Face
	port *Port
	loc  ethLocator
	priv *C.EthFacePriv
	rxf  *rxFlow
}

func (face *ethFace) ptr() *C.Face {
	return (*C.Face)(face.Ptr())
}

// New creates a face on the given port.
func New(port *Port, loc ethLocator) (iface.Face, error) {
	for _, f := range port.faces {
		if f.loc.conflictsWith(loc) {
			return nil, LocatorConflictError{a: loc, b: f.loc}
		}
	}

	face := &ethFace{
		port: port,
		loc:  loc,
	}
	return iface.New(iface.NewParams{
		Config:     port.cfg.Config,
		Socket:     port.dev.NumaSocket(),
		SizeofPriv: uintptr(C.sizeof_EthFacePriv),
		TxHeadroom: int(C.ETHHDR_BUFLEN),
		Init: func(f iface.Face) error {
			face.Face = f
			c := face.ptr()
			c.txBurstOp = (C.FaceImpl_TxBurst)(C.EthFace_TxBurst)

			priv := (*C.EthFacePriv)(C.Face_GetPriv(c))
			*priv = C.EthFacePriv{
				port:   C.uint16_t(port.dev.ID()),
				faceID: C.FaceID(f.ID()),
			}
			cLoc := face.loc.cLoc()
			priv.hdrLen = C.EthLocator_MakeTxHdr(cLoc.ptr(), &priv.txHdr[0])
			priv.rxMatch = C.EthLocator_MakeRxMatch(cLoc.ptr(), &priv.rxMatchBuffer[0])

			face.priv = priv
			return nil
		},
		Start: func(iface.Face) (iface.Face, error) {
			return face, port.startFace(face, false)
		},
		Locator: func(iface.Face) iface.Locator {
			return face.loc
		},
		Stop: func(iface.Face) error {
			return face.port.stopFace(face)
		},
		Close: func(iface.Face) error {
			if len(face.port.faces) == 0 {
				face.port.Close()
			}
			return nil
		},
	})
}
