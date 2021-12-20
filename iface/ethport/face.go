package ethport

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"errors"
	"net"

	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"
)

// FaceConfig contains additional face configuration.
// They appear as input-only fields of EtherLocator.
type FaceConfig struct {
	iface.Config

	// EthDev causes the face to be created on a specific Ethernet adapter.
	// This allows setting a local MAC address that differs from the physical MAC address.
	//
	// If omitted, local MAC address is used to find the Ethernet adapter.
	//
	// In either case, a Port must be created on the Ethernet adapter before creating faces.
	EthDev ethdev.EthDev `json:"-"`

	// Port is GraphQL ID of the EthDev.
	// This field has the same semantics as EthDev.
	// If both EthDev and Port are specified, EthDev takes priority.
	Port string `json:"port,omitempty"`

	// MaxRxQueues is the maximum number of RX queues for this face.
	// It is meaningful only if the port is using RxFlow.
	// For most DPDK drivers, it is effective in improving performance on VXLAN face only.
	//
	// Default is 1.
	MaxRxQueues int `json:"maxRxQueues,omitempty"`

	// DisableTxMultiSegOffload forces every packet to be copied into a linear buffer in software.
	DisableTxMultiSegOffload bool `json:"disableTxMultiSegOffload,omitempty"`

	// DisableTxChecksumOffload disables the usage of IPv4 and UDP checksum offloads.
	DisableTxChecksumOffload bool `json:"disableTxChecksumOffload,omitempty"`

	// TxChecksumRequireLinear indicates software TX checksum requires a linear buffer.
	// This is set for UDP over IPv6, which requires UDP checksum but rte_ipv6_udptcp_cksum expects a linear buffer.
	TxChecksumRequireLinear bool `json:"-"`

	// privFaceConfig is hidden from JSON output.
	privFaceConfig *FaceConfig
}

// EthFaceConfig implements Locator interface.
func (cfg FaceConfig) EthFaceConfig() FaceConfig {
	if cfg.privFaceConfig != nil {
		return *cfg.privFaceConfig
	}
	return cfg
}

// HideFaceConfigFromJSON hides FaceConfig fields from JSON marshaling.
func (cfg *FaceConfig) HideFaceConfigFromJSON() {
	copy := *cfg
	*cfg = FaceConfig{privFaceConfig: &copy}
}

// FindPort finds an existing Port as cfg.EthDev, cfg.Port, or local MAC address.
func (cfg FaceConfig) FindPort(local net.HardwareAddr) (port *Port, e error) {
	dev := cfg.EthDev
	switch {
	case dev != nil:
	case cfg.Port != "":
		gqlserver.RetrieveNodeOfType(ethdev.GqlEthDevNodeType, cfg.Port, &dev)
	default:
		dev = ethdev.FromHardwareAddr(local)
	}

	port = Find(dev)
	if port == nil {
		return nil, errors.New("Port does not exist; Port must be created before creating face")
	}
	return port, nil
}

// Face represents a face on Ethernet Port.
type Face struct {
	iface.Face
	port   *Port
	loc    Locator
	cLoc   CLocator
	logger *zap.Logger
	priv   *C.EthFacePriv
	flow   *C.struct_rte_flow
	rxf    []*rxgFlow
}

// NewFace creates a face on the given port.
func NewFace(port *Port, loc Locator) (iface.Face, error) {
	face := &Face{
		port:   port,
		loc:    loc,
		cLoc:   loc.EthCLocator(),
		logger: port.logger,
	}
	port, loc = nil, nil
	return iface.New(iface.NewParams{
		Config:     face.loc.EthFaceConfig().Config.WithMaxMTU(face.port.cfg.MTU + C.RTE_ETHER_HDR_LEN - face.cLoc.sizeofHeader()),
		Socket:     face.port.dev.NumaSocket(),
		SizeofPriv: uintptr(C.sizeof_EthFacePriv),
		Init: func(f iface.Face) (iface.InitResult, error) {
			for _, other := range face.port.faces {
				if !face.cLoc.canCoexist(other.cLoc) {
					return iface.InitResult{}, LocatorConflictError{a: face.loc, b: other.loc}
				}
			}

			face.Face = f
			faceC := (*C.Face)(face.Ptr())
			id := f.ID()
			face.logger = face.logger.With(id.ZapField("id"))

			face.priv = (*C.EthFacePriv)(C.Face_GetPriv(faceC))
			*face.priv = C.EthFacePriv{
				port:   C.uint16_t(face.port.dev.ID()),
				faceID: C.FaceID(id),
			}

			cfg := face.loc.EthFaceConfig()
			C.EthRxMatch_Prepare(&face.priv.rxMatch, face.cLoc.ptr())
			useTxMultiSegOffload := !cfg.DisableTxMultiSegOffload && face.port.devInfo.HasTxMultiSegOffload()
			useTxChecksumOffload := !cfg.DisableTxChecksumOffload && face.port.devInfo.HasTxChecksumOffload()
			if !useTxChecksumOffload && cfg.TxChecksumRequireLinear {
				useTxMultiSegOffload = false
			}
			C.EthTxHdr_Prepare(&face.priv.txHdr, face.cLoc.ptr(), C.bool(useTxChecksumOffload))

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
			if len(face.port.faces) == 0 && face.port.cfg.AutoClose {
				return face.port.Close()
			}
			return nil
		},
	})
}
