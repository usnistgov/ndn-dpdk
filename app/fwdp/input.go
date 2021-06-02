package fwdp

/*
#include "../../csrc/fwdp/fwd.h"
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// Input represents an input thread.
type Input struct {
	id  int
	rxl iface.RxLoop
}

// Init initializes the input thread.
func (fwi *Input) Init(lc eal.LCore, ndt *ndt.Ndt, fwds []*Fwd) error {
	socket := lc.NumaSocket()

	fwi.rxl = iface.NewRxLoop(socket)
	fwi.rxl.SetLCore(lc)

	demuxI := fwi.rxl.InterestDemux()
	demuxI.InitNdt(ndt.Threads()[fwi.id])
	demuxD := fwi.rxl.DataDemux()
	demuxD.InitToken(uint8(C.FwTokenOffsetFwdID))
	demuxN := fwi.rxl.NackDemux()
	demuxN.InitToken(uint8(C.FwTokenOffsetFwdID))
	for i, fwd := range fwds {
		demuxI.SetDest(i, fwd.queueI)
		demuxD.SetDest(i, fwd.queueD)
		demuxN.SetDest(i, fwd.queueN)
	}

	return nil
}

// Close stops the thread.
func (fwi *Input) Close() error {
	return fwi.rxl.Close()
}

func (fwi *Input) String() string {
	return fmt.Sprintf("input%d", fwi.id)
}

// Thread implements ealthread.WithThread interface.
func (fwi *Input) Thread() ealthread.Thread {
	return fwi.rxl
}

func newInput(id int) *Input {
	return &Input{id: id}
}
