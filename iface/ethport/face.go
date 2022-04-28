package ethport

/*
#include "../../csrc/ethface/face.h"
*/
import "C"
import (
	"errors"
	"net"

	"github.com/usnistgov/ndn-dpdk/core/macaddr"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/iface"
	"go.uber.org/zap"
)

// FaceConfig contains additional face configuration.
// They appear as input-only fields of ethface.EtherLocator.
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

	// NRxQueues is the number of RX queues for this face.
	// It is meaningful only if the port is using RxFlow.
	// For most DPDK drivers, it is effective in improving performance on VXLAN face only.
	//
	// Default is 1.
	NRxQueues int `json:"nRxQueues,omitempty"`

	// DisableTxMultiSegOffload forces every packet to be copied into a linear buffer in software.
	DisableTxMultiSegOffload bool `json:"disableTxMultiSegOffload,omitempty"`

	// DisableTxChecksumOffload disables the usage of IPv4 and UDP checksum offloads.
	DisableTxChecksumOffload bool `json:"disableTxChecksumOffload,omitempty"`

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

// FindPort finds an existing Port from cfg.EthDev, cfg.Port, or local MAC address.
func (cfg FaceConfig) FindPort(local net.HardwareAddr) (port *Port, e error) {
	dev := cfg.EthDev
	switch {
	case dev != nil:
	case cfg.Port != "":
		dev = ethdev.GqlEthDevType.Retrieve(cfg.Port)
	case macaddr.IsUnicast(local):
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
	logger *zap.Logger
	priv   *C.EthFacePriv

	flow *C.struct_rte_flow
	rxf  []*rxgFlow
}

// NewFace creates a face on the given port.
func NewFace(port *Port, loc Locator) (iface.Face, error) {
	face := &Face{
		port:   port,
		loc:    loc,
		logger: port.logger,
	}
	port, loc = nil, nil
	return iface.New(iface.NewParams{
		Config:     face.loc.EthFaceConfig().Config.WithMaxMTU(face.port.cfg.MTU - NewTxHdr(face.loc, false).IPLen()),
		Socket:     face.port.dev.NumaSocket(),
		SizeofPriv: C.sizeof_EthFacePriv,
		Init: func(f iface.Face) (iface.InitResult, error) {
			face.port.mutex.Lock()
			defer face.port.mutex.Unlock()

			for _, other := range face.port.faces {
				if e := CheckLocatorCoexist(face.loc, other.loc); e != nil {
					return iface.InitResult{}, e
				}
			}

			face.Face = f
			id, faceC := face.ID(), (*C.Face)(face.Ptr())
			face.logger = face.logger.With(id.ZapField("id"))

			face.priv = (*C.EthFacePriv)(C.Face_GetPriv(faceC))
			*face.priv = C.EthFacePriv{
				port:   C.uint16_t(face.port.dev.ID()),
				faceID: C.FaceID(id),
			}

			cfg := face.loc.EthFaceConfig()
			NewRxMatch(face.loc).copyToC(&face.priv.rxMatch)
			useTxMultiSegOffload := !cfg.DisableTxMultiSegOffload && face.port.devInfo.HasTxMultiSegOffload()
			useTxChecksumOffload := !cfg.DisableTxChecksumOffload && face.port.devInfo.HasTxChecksumOffload()
			NewTxHdr(face.loc, useTxChecksumOffload).copyToC(&face.priv.txHdr)

			return iface.InitResult{
				Face:        face,
				TxLinearize: !useTxMultiSegOffload,
				TxBurst:     C.EthFace_TxBurst,
			}, nil
		},
		Start: func() error {
			face.port.mutex.Lock()
			defer face.port.mutex.Unlock()

			id := face.ID()
			if e := face.port.rxImpl.Start(face); e != nil {
				face.logger.Error("face start error; change Port config or locator, and try again", zap.Error(e))
				return e
			}
			ethnetif.XDPInsertFaceMapEntry(face.port.dev, face.loc.EthCLocator().toXDP(), 0)

			face.port.activateTx(face)
			face.logger.Info("face started")
			face.port.faces[id] = face
			return nil
		},
		Locator: func() iface.Locator {
			return face.loc
		},
		Stop: func() error {
			face.port.mutex.Lock()
			defer face.port.mutex.Unlock()

			id := face.ID()
			delete(face.port.faces, id)

			ethnetif.XDPDeleteFaceMapEntry(face.port.dev, face.loc.EthCLocator().toXDP())
			if e := face.port.rxImpl.Stop(face); e != nil {
				face.logger.Warn("face stop error", zap.Error(e))
			} else {
				face.logger.Info("face stopped")
			}
			face.port.deactivateTx(face)
			return nil
		},
		Close: func() error {
			if face.port.cfg.AutoClose {
				face.port.mutex.Lock()
				nFaces := len(face.port.faces)
				face.port.mutex.Unlock()
				if nFaces == 0 {
					return face.port.Close()
				}
			}
			return nil
		},
	})
}
