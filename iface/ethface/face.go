// Package ethface implements Ethernet faces using DPDK Ethernet devices.
package ethface

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/iface"
)

var logger = logging.New("ethface")

type ethFace struct {
	iface.Face
	port *Port
	loc  ethLocator
	cloc cLocator
	priv *C.EthFacePriv
	flow *C.struct_rte_flow
	rxf  []*rxFlow
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
		Config:     loc.faceConfig().Config.WithMaxMTU(port.cfg.MTU + C.RTE_ETHER_HDR_LEN - face.cloc.sizeofHeader()),
		Socket:     port.dev.NumaSocket(),
		SizeofPriv: uintptr(C.sizeof_EthFacePriv),
		Init: func(f iface.Face) (iface.InitResult, error) {
			for _, other := range port.faces {
				if !face.cloc.canCoexist(other.cloc) {
					return iface.InitResult{}, LocatorConflictError{a: loc, b: other.loc}
				}
			}

			face.Face = f
			faceC := face.ptr()

			priv := (*C.EthFacePriv)(C.Face_GetPriv(faceC))
			*priv = C.EthFacePriv{
				port:   C.uint16_t(port.dev.ID()),
				faceID: C.FaceID(f.ID()),
			}

			cfg, devInfo := loc.faceConfig(), port.dev.DevInfo()

			C.EthRxMatch_Prepare(&priv.rxMatch, face.cloc.ptr())
			useTxMultiSegOffload := !cfg.DisableTxMultiSegOffload && devInfo.HasTxMultiSegOffload()
			useTxChecksumOffload := !cfg.DisableTxChecksumOffload && devInfo.HasTxChecksumOffload()
			if loc, ok := face.loc.(UDPLocator); ok && !useTxChecksumOffload && loc.RemoteIP.Unmap().Is6() {
				// UDP checksum is required in IPv6, and rte_ipv6_udptcp_cksum expects a linear buffer
				useTxMultiSegOffload = false
			}
			C.EthTxHdr_Prepare(&priv.txHdr, face.cloc.ptr(), C.bool(useTxChecksumOffload))

			face.priv = priv
			return iface.InitResult{
				Face:        face,
				L2TxBurst:   C.EthFace_TxBurst,
				TxLinearize: !useTxMultiSegOffload,
			}, nil
		},
		Start: func() error {
			return port.startFace(face, false)
		},
		Locator: func() iface.Locator {
			return face.loc
		},
		Stop: func() error {
			return face.port.stopFace(face)
		},
		Close: func() error {
			if len(face.port.faces) == 0 {
				return face.port.Close()
			}
			return nil
		},
	})
}
