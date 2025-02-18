package ethport

/*
#include "../../csrc/ethface/passthru.h"
*/
import "C"
import (
	"fmt"
	"math"
	"sync"
	"unsafe"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
	"go.uber.org/zap"
)

// SchemePassthru indicates a pass-through face.
const SchemePassthru = "passthru"

// MakePassthruTapName constructs TAP netif name for a passthru face on ethdev.
func MakePassthruTapName(dev ethdev.EthDev) string {
	return fmt.Sprintf("ndndpdkPT%d", dev.ID())
}

// GtpipFromPassthruFace retrieves GTP-IP table associated with a pass-through face.
func GtpipFromPassthruFace(id iface.ID) *Gtpip {
	passthruPortsMutex.Lock()
	defer passthruPortsMutex.Unlock()

	if fport, ok := passthruPorts[id]; ok {
		return fport.gtpip
	}
	return nil
}

var (
	passthruPorts      = map[iface.ID]*passthruPort{}
	passthruPortsMutex sync.Mutex
)

// passthruPort holds a passthru face and the associated TAP netif.
type passthruPort struct {
	face   *Face
	tapDev ethdev.EthDev
	gtpip  *Gtpip
}

var (
	_ iface.RxGroup           = (*passthruPort)(nil)
	_ iface.RxGroupSingleFace = (*passthruPort)(nil)
)

func (fport *passthruPort) NumaSocket() eal.NumaSocket {
	return fport.tapDev.NumaSocket()
}

func (fport *passthruPort) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(&fport.face.passthruC().base),
		fmt.Sprintf("EthRxPassthru(face=%d,port=%d)", fport.face.ID(), fport.face.port.EthDev().ID())
}

func (fport *passthruPort) Faces() []iface.Face {
	return []iface.Face{fport.face}
}

func (passthruPort) RxGroupIsSingleFace() {}

func (fport *passthruPort) startTap() (e error) {
	dev := fport.face.port.dev
	if fport.tapDev, e = ethdev.NewTap(MakePassthruTapName(dev), dev.HardwareAddr()); e != nil {
		return e
	}

	var cfg ethdev.Config
	cfg.MTU = dev.MTU()
	cfg.AddRxQueues(1, ethdev.RxQueueConfig{RxPool: ndni.PacketMempool.Get(fport.face.NumaSocket())})
	cfg.AddTxQueues(1, ethdev.TxQueueConfig{})
	if e := fport.tapDev.Start(cfg); e != nil {
		fport.tapDev.Close()
		return e
	}

	ptC := fport.face.passthruC()
	ptC.tapPort = C.uint16_t(fport.tapDev.ID())
	ptC.base.rxBurst = C.RxGroup_RxBurstFunc(C.EthPassthru_TapPortRxBurst)

	rxl := iface.ActivateRxGroup(fport)
	fport.face.logger.Info("activated TAP device",
		fport.tapDev.ZapField("tap-port"),
		rxl.LCore().ZapField("tap-rxl-lc"),
	)

	return nil
}

func (fport *passthruPort) stopTap() {
	tapDevField := fport.tapDev.ZapField("tap-port")
	iface.DeactivateRxGroup(fport)
	fport.face.logger.Info("deactivated TAP device",
		tapDevField,
	)

	if e := fport.tapDev.Close(); e != nil {
		fport.face.logger.Error("close TAP device error",
			tapDevField,
			zap.Error(e),
		)
	}
}

func (fport *passthruPort) enableGtpip(cfg GtpipConfig) (e error) {
	fport.gtpip, e = NewGtpip(cfg, fport.NumaSocket())

	if e == nil {
		ptC := fport.face.passthruC()
		ptC.gtpip = (*C.EthGtpip)(fport.gtpip)
	}

	return
}

func (fport *passthruPort) disableGtpip() {
	if fport.gtpip == nil {
		return
	}
	fport.gtpip.Close()
	fport.gtpip = nil
}

func passthruInit(face *Face, initResult *iface.InitResult) {
	ptC := face.passthruC()
	ptC.tapPort = math.MaxUint16
	ptC.gtpip = nil
	initResult.RxInput = C.EthPassthru_FaceRxInput
	initResult.TxLoop = C.EthPassthru_TxLoop
}

func passthruStart(face *Face) error {
	passthruPortsMutex.Lock()
	defer passthruPortsMutex.Unlock()

	fport := &passthruPort{face: face}
	if e := fport.startTap(); e != nil {
		return e
	}

	type withGtpipConfig interface {
		GtpipConfig() *GtpipConfig
	}
	if loc, ok := face.loc.(withGtpipConfig); ok {
		if cfg := loc.GtpipConfig(); cfg != nil {
			if e := fport.enableGtpip(*cfg); e != nil {
				fport.stopTap()
				return e
			}
		}
	}

	passthruPorts[face.ID()] = fport
	return nil
}

func passthruStop(face *Face) {
	passthruPortsMutex.Lock()
	defer passthruPortsMutex.Unlock()

	fport := passthruPorts[face.ID()]
	fport.disableGtpip()
	fport.stopTap()
	delete(passthruPorts, face.ID())
}
