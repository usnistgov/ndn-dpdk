package fwdp

/*
#include "input.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"ndn-dpdk/appinit"
	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type InputBase struct {
	appinit.ThreadBase
	id int
	c  *C.FwInput
}

func (fwi *InputBase) Init(ndt *ndt.Ndt, fwds []*Fwd) error {
	numaSocket := fwi.GetNumaSocket()

	fwi.c = C.FwInput_New((*C.Ndt)(ndt.GetPtr()), C.uint8_t(fwi.id),
		C.uint8_t(len(fwds)), C.unsigned(numaSocket))
	if fwi.c == nil {
		return dpdk.GetErrno()
	}

	for _, fwd := range fwds {
		C.FwInput_Connect(fwi.c, fwd.c)
	}

	return nil
}

func (fwi *InputBase) Close() error {
	dpdk.Free(fwi.c)
	return nil
}

type Input struct {
	InputBase
	rxl iface.IRxLooper
}

func newInput(id int) *Input {
	var fwi Input
	fwi.ResetThreadBase()
	fwi.id = id
	return &fwi
}

func (fwi *Input) String() string {
	return fmt.Sprintf("input%d", fwi.id)
}

func (fwi *Input) Launch() error {
	if fwi.rxl == nil {
		return errors.New("rxl is unset")
	}
	return fwi.LaunchImpl(func() int {
		const burstSize = 64
		fwi.rxl.RxLoop(burstSize, unsafe.Pointer(C.FwInput_FaceRx), unsafe.Pointer(fwi.c))
		return 0
	})
}

func (fwi *Input) Stop() error {
	return fwi.StopImpl(appinit.NewStopRxLooper(fwi.rxl))
}
