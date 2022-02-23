package fwdp

/*
#include "../../csrc/fwdp/token.h"
*/
import "C"
import (
	"fmt"

	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

type demuxPreparer struct {
	Ndt  *ndt.Ndt
	Fwds []*Fwd
}

func (p *demuxPreparer) PrepareDemuxI(demux *iface.InputDemux, socket eal.NumaSocket) {
	ndq := demux.InitNdt()
	ndq.Init(p.Ndt, socket)
	for i, fwd := range p.Fwds {
		demux.SetDest(i, fwd.queueI)
	}
}

func (p *demuxPreparer) PrepareDemuxD(demux *iface.InputDemux) {
	demux.InitToken(C.FwTokenOffsetFwdID)
	for i, fwd := range p.Fwds {
		demux.SetDest(i, fwd.queueD)
	}
}

func (p *demuxPreparer) PrepareDemuxN(demux *iface.InputDemux) {
	demux.InitToken(C.FwTokenOffsetFwdID)
	for i, fwd := range p.Fwds {
		demux.SetDest(i, fwd.queueN)
	}
}

// Input represents an input thread.
type Input struct {
	id  int
	rxl iface.RxLoop
}

// Init initializes the input thread.
func (fwi *Input) Init(lc eal.LCore, demuxPrep *demuxPreparer) error {
	socket := lc.NumaSocket()

	fwi.rxl = iface.NewRxLoop(socket)
	fwi.rxl.SetLCore(lc)

	demuxPrep.PrepareDemuxI(fwi.rxl.DemuxOf(ndni.PktInterest), socket)
	demuxPrep.PrepareDemuxD(fwi.rxl.DemuxOf(ndni.PktData))
	demuxPrep.PrepareDemuxN(fwi.rxl.DemuxOf(ndni.PktNack))

	return nil
}

// Close stops the thread.
func (fwi *Input) Close() error {
	return fwi.rxl.Close()
}

func (fwi *Input) String() string {
	return fmt.Sprintf("input%d", fwi.id)
}

func newInput(id int) *Input {
	return &Input{id: id}
}
