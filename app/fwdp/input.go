package fwdp

/*
#include "input.h"
*/
import "C"

import (
	"fmt"
	"unsafe"

	"ndn-dpdk/container/ndt"
	"ndn-dpdk/dpdk"
	"ndn-dpdk/iface"
)

type InputBase struct {
	id int
	c  *C.FwInput
}

func (fwi *InputBase) Init(ndt *ndt.Ndt, fwds []*Fwd, numaSocket dpdk.NumaSocket) error {
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
	lc  dpdk.LCore
	rxl *iface.RxLoop
}

func newInput(id int, lc dpdk.LCore) *Input {
	var fwi Input
	fwi.id = id
	fwi.lc = lc
	return &fwi
}

func (fwi *Input) String() string {
	return fmt.Sprintf("input%d", fwi.id)
}

func (fwi *Input) prepareLaunch(rxl *iface.RxLoop) {
	fwi.rxl = rxl
	rxl.SetCallback(unsafe.Pointer(C.FwInput_FaceRx), unsafe.Pointer(fwi.c))
}

func (fwi *Input) Stop() {
	if fwi.rxl == nil {
		return
	}
	fwi.rxl.Stop()
}
