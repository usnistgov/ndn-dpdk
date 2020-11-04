// Package ethface implements Ethernet faces using DPDK Ethernet devices.
package ethface

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"errors"
	"net"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Error conditions.
var (
	ErrLocalMismatch          = errors.New("port has a different local address")
	ErrRemoteDuplicateGroup   = errors.New("port has another face with a group address")
	ErrRemoteDuplicateUnicast = errors.New("port has another face with same unicast address")
	ErrRemoteInvalid          = errors.New("invalid MAC address")
)

type ethLocator interface {
	iface.Locator

	local() net.HardwareAddr
	remote() net.HardwareAddr
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

// New creates an Ethernet face on the given port.
func New(port *Port, loc ethLocator) (iface.Face, error) {
	if !macaddr.Equal(loc.local(), port.local) {
		return nil, ErrLocalMismatch
	}

	remote := loc.remote()
	switch {
	case macaddr.IsMulticast(remote):
		if face := port.FindFace(nil); face != nil {
			return nil, ErrRemoteDuplicateGroup
		}
	case macaddr.IsUnicast(remote):
		if face := port.FindFace(remote); face != nil {
			return nil, ErrRemoteDuplicateUnicast
		}
	default:
		return nil, ErrRemoteInvalid
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
			if face.port.CountFaces() == 0 {
				face.port.Close()
			}
			return nil
		},
	})
}
