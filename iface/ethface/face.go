// Package ethface implements Ethernet faces using DPDK Ethernet devices.
package ethface

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/iface"
)

type ethFace struct {
	iface.Face
	port *Port
	loc  ethLocator
	cloc cLocator
	priv *C.EthFacePriv
	rxf  *rxFlow
}

func (face *ethFace) ptr() *C.Face {
	return (*C.Face)(face.Ptr())
}

// New creates a face on the given port.
func New(port *Port, loc ethLocator) (iface.Face, error) {
	face := &ethFace{
		port: port,
		loc:  loc,
		cloc: loc.cLoc(),
	}
	return iface.New(iface.NewParams{
		Config:           port.cfg.Config,
		Socket:           port.dev.NumaSocket(),
		SizeofPriv:       uintptr(C.sizeof_EthFacePriv),
		TxHeadroom:       int(C.ETHHDR_MAXLEN),
		TxHeaderOverhead: face.cloc.sizeofHeader(),
		Init: func(f iface.Face) error {
			for _, other := range port.faces {
				if !face.cloc.canCoexist(other.cloc) {
					return LocatorConflictError{a: loc, b: other.loc}
				}
			}

			face.Face = f
			c := face.ptr()
			c.txBurstOp = (C.FaceImpl_TxBurst)(C.EthFace_TxBurst)

			priv := (*C.EthFacePriv)(C.Face_GetPriv(c))
			*priv = C.EthFacePriv{
				port:   C.uint16_t(port.dev.ID()),
				faceID: C.FaceID(f.ID()),
			}

			C.EthRxMatch_Prepare(&priv.rxMatch, face.cloc.ptr())
			C.EthTxHdr_Prepare(&priv.txHdr, face.cloc.ptr(), C.bool(port.dev.HasChecksumOffloads() && !port.cfg.DisableTxOffloads))
			if priv.rxMatch.len != priv.txHdr.len {
				// assertion needed by EthFace_FlowRxBurst function
				panic(fmt.Sprintf("assert(priv.rxMatch.len==priv.txHdr.len) failed: %d %d", priv.rxMatch.len, priv.txHdr.len))
			}

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
