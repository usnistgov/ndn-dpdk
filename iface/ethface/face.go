// Package ethface implements Ethernet faces using DPDK Ethernet devices.
package ethface

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/core/logging"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"
)

var logger = logging.New("ethface")

type ethFace struct {
	iface.Face
	port   *Port
	loc    ethLocator
	cloc   cLocator
	logger *zap.Logger
	priv   *C.EthFacePriv
	flow   *C.struct_rte_flow
	rxf    []*rxFlow
}

func (face *ethFace) ptr() *C.Face {
	return (*C.Face)(face.Ptr())
}

// New creates a face on the given port.
func New(port *Port, loc ethLocator) (iface.Face, error) {
	face := &ethFace{
		port:   port,
		loc:    loc,
		cloc:   loc.cLoc(),
		logger: port.logger,
	}
	port, loc = nil, nil
	return iface.New(iface.NewParams{
		Config:     face.loc.faceConfig().Config.WithMaxMTU(face.port.cfg.MTU + C.RTE_ETHER_HDR_LEN - face.cloc.sizeofHeader()),
		Socket:     face.port.dev.NumaSocket(),
		SizeofPriv: uintptr(C.sizeof_EthFacePriv),
		Init: func(f iface.Face) (iface.InitResult, error) {
			for _, other := range face.port.faces {
				if !face.cloc.canCoexist(other.cloc) {
					return iface.InitResult{}, LocatorConflictError{a: face.loc, b: other.loc}
				}
			}

			face.Face = f
			faceC := face.ptr()
			id := f.ID()
			face.logger = face.logger.With(id.ZapField("id"))

			priv := (*C.EthFacePriv)(C.Face_GetPriv(faceC))
			*priv = C.EthFacePriv{
				port:   C.uint16_t(face.port.dev.ID()),
				faceID: C.FaceID(id),
			}

			cfg, devInfo := face.loc.faceConfig(), face.port.dev.DevInfo()

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
			id := face.ID()
			if e := face.port.rxImpl.Start(face); e != nil {
				face.logger.Error("face start error; change Port config or locator, and try again", zap.Error(e))
				return e
			}

			face.port.activateTx(face)
			face.logger.Info("face started")
			face.port.faces[id] = face
			return nil
		},
		Locator: func() iface.Locator {
			return face.loc
		},
		Stop: func() error {
			id := face.ID()
			delete(face.port.faces, id)
			if e := face.port.rxImpl.Stop(face); e != nil {
				face.logger.Warn("face stop error", zap.Error(e))
			} else {
				face.logger.Info("face stopped")
			}
			face.port.deactivateTx(face)
			return nil
		},
		Close: func() error {
			if len(face.port.faces) == 0 && face.port.autoClose {
				return face.port.Close()
			}
			return nil
		},
	})
}
