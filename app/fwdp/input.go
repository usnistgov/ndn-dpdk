package fwdp

/*
#include "../../csrc/fwdp/token.h"
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// Input represents an input thread.
type Input struct {
	id  int
	rxl iface.RxLoop
}

var _ DispatchThread = (*Input)(nil)

// DispatchThreadID implements DispatchThread interface.
func (fwi *Input) DispatchThreadID() int {
	return fwi.id
}

func (fwi *Input) String() string {
	return fmt.Sprintf("input%d", fwi.id)
}

// DemuxOf implements DispatchThread interface.
func (fwi *Input) DemuxOf(t ndni.PktType) *iface.InputDemux {
	return fwi.rxl.DemuxOf(t)
}

// Close stops the thread.
func (fwi *Input) Close() error {
	return fwi.rxl.Close()
}

// newInput creates an input thread.
func newInput(id int, lc eal.LCore, demuxPrep *demuxPreparer) (fwi *Input, e error) {
	socket := lc.NumaSocket()
	fwi = &Input{
		id:  id,
		rxl: iface.NewRxLoop(socket),
	}
	fwi.rxl.SetLCore(lc)

	demuxPrep.Prepare(fwi, socket)
	return fwi, nil
}
