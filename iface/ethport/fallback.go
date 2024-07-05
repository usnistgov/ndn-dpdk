package ethport

/*
#include "../../csrc/ethface/fallback.h"
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

func MakeFallbackTapName(dev ethdev.EthDev) string {
	return fmt.Sprintf("ndndpdk-f-%d", dev.ID())
}

var fallbackPorts = map[iface.ID]*fallbackPort{}

type fallbackPort struct {
	face   *Face
	tapDev ethdev.EthDev
}

var (
	_ iface.RxGroup           = (*fallbackPort)(nil)
	_ iface.RxGroupSingleFace = (*fallbackPort)(nil)
)

func (fport *fallbackPort) NumaSocket() eal.NumaSocket {
	return fport.tapDev.NumaSocket()
}

func (fport *fallbackPort) RxGroup() (ptr unsafe.Pointer, desc string) {
	return unsafe.Pointer(&fport.face.priv.rxf[0].base),
		fmt.Sprintf("EthRxFallback(face=%d,port=%d)", fport.face.ID(), fport.face.port.EthDev().ID())
}

func (fport *fallbackPort) Faces() []iface.Face {
	return []iface.Face{fport.face}
}

func (fallbackPort) RxGroupIsSingleFace() {}

func (fport *fallbackPort) startTap() (e error) {
	dev := fport.face.port.dev
	if fport.tapDev, e = ethdev.NewTap(MakeFallbackTapName(dev), dev.HardwareAddr()); e != nil {
		return e
	}

	var cfg ethdev.Config
	cfg.AddRxQueues(1, ethdev.RxQueueConfig{RxPool: ndni.PacketMempool.Get(fport.face.NumaSocket())})
	cfg.AddTxQueues(1, ethdev.TxQueueConfig{})
	if e := fport.tapDev.Start(cfg); e != nil {
		fport.tapDev.Close()
		return e
	}

	priv := fport.face.priv
	priv.tapPort = C.uint16_t(fport.tapDev.ID())
	rxfC := &priv.rxf[0]
	rxfC.base.rxBurst = C.RxGroup_RxBurstFunc(C.EthFallback_TapPortRxBurst)
	rxfC.port, rxfC.queue, rxfC.faceID = fport.face.priv.tapPort, 0, priv.faceID

	rxl := iface.ActivateRxGroup(fport)
	fport.face.logger.Info("activated TAP device",
		fport.tapDev.ZapField("tap-port"),
		rxl.LCore().ZapField("tap-rxl-lc"),
	)

	return nil
}

func (fport *fallbackPort) stopTap() {
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

func fallbackInit(face *Face, initResult *iface.InitResult) {
	_ = face
	initResult.RxInput = C.EthFallback_FaceRxInput
	initResult.TxLoop = C.EthFallback_TxLoop
}

func fallbackStart(face *Face) error {
	fport := &fallbackPort{face: face}
	if e := fport.startTap(); e != nil {
		return e
	}
	fallbackPorts[face.ID()] = fport
	return nil
}

func fallbackStop(face *Face) {
	fport := fallbackPorts[face.ID()]
	fport.stopTap()
	delete(fallbackPorts, face.ID())
}
