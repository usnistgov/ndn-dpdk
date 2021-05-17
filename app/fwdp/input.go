package fwdp

/*
#include "../../csrc/fwdp/fwd.h"
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Input represents an input thread.
type Input struct {
	id  int
	rxl iface.RxLoop
}

func newInput(id int, lc eal.LCore, ndt *ndt.Ndt, fwds []*Fwd) *Input {
	socket := lc.NumaSocket()
	var fwi Input
	fwi.id = id

	rxl := iface.NewRxLoop(socket)
	rxl.SetLCore(lc)

	demuxI := rxl.InterestDemux()
	demuxI.InitNdt(ndt.Threads()[id])
	demuxD := rxl.DataDemux()
	demuxD.InitToken(uint8(C.FwTokenOffsetFwdID))
	demuxN := rxl.NackDemux()
	demuxN.InitToken(uint8(C.FwTokenOffsetFwdID))
	for i, fwd := range fwds {
		demuxI.SetDest(i, fwd.queueI)
		demuxD.SetDest(i, fwd.queueD)
		demuxN.SetDest(i, fwd.queueN)
	}

	fwi.rxl = rxl
	return &fwi
}

// Close stops the thread.
func (fwi *Input) Close() error {
	return fwi.rxl.Close()
}

func (fwi *Input) String() string {
	return fmt.Sprintf("input%d", fwi.id)
}
