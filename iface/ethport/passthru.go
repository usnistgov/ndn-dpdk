package ethport

/*
#include "../../csrc/ethface/passthru.h"
*/
import "C"
import (
	"fmt"
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

var passthruPorts = map[iface.ID]*passthruPort{}

// passthruPort holds a passthru face and the associated TAP netif.
type passthruPort struct {
	face   *Face
	tapDev ethdev.EthDev
}

var (
	_ iface.RxGroup           = (*passthruPort)(nil)
	_ iface.RxGroupSingleFace = (*passthruPort)(nil)
)

func (fport *passthruPort) NumaSocket() eal.NumaSocket {
	return fport.tapDev.NumaSocket()
}

func (fport *passthruPort) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(&fport.face.priv.rxf[0].base),
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

	priv := fport.face.priv
	priv.tapPort = C.uint16_t(fport.tapDev.ID())
	rxfC := &priv.rxf[0]
	rxfC.base.rxBurst = C.RxGroup_RxBurstFunc(C.EthPassthru_TapPortRxBurst)
	rxfC.port, rxfC.queue, rxfC.faceID = fport.face.priv.tapPort, 0, priv.faceID

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
	fport.face.logger.Info("deactivate TAP device",
		tapDevField,
	)

	if e := fport.tapDev.Close(); e != nil {
		fport.face.logger.Error("close TAP device error",
			tapDevField,
			zap.Error(e),
		)
	}
}

func passthruInit(face *Face, initResult *iface.InitResult) {
	_ = face
	initResult.RxInput = C.EthPassthru_FaceRxInput
	initResult.TxLoop = C.EthPassthru_TxLoop
}

func passthruStart(face *Face) error {
	fport := &passthruPort{face: face}
	if e := fport.startTap(); e != nil {
		return e
	}
	passthruPorts[face.ID()] = fport
	return nil
}

func passthruStop(face *Face) {
	fport := passthruPorts[face.ID()]
	fport.stopTap()
	delete(passthruPorts, face.ID())
}
