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
)

type demuxPreparer struct {
	Fwds        []*Fwd
	NdtQueriers []*ndt.Querier
}

func (p *demuxPreparer) PrepareDemuxI(id int, demux *iface.InputDemux) {
	ndq := p.NdtQueriers[id]
	if ndq == nil {
		panic("duplicate NDT querier ID")
	}
	p.NdtQueriers[id] = nil

	demux.InitNdt(ndq)
	for i, fwd := range p.Fwds {
		demux.SetDest(i, fwd.queueI)
	}
}

func (p *demuxPreparer) PrepareDemuxD(demux *iface.InputDemux) {
	demux.InitToken(uint8(C.FwTokenOffsetFwdID))
	for i, fwd := range p.Fwds {
		demux.SetDest(i, fwd.queueD)
	}
}

func (p *demuxPreparer) PrepareDemuxN(demux *iface.InputDemux) {
	demux.InitToken(uint8(C.FwTokenOffsetFwdID))
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

	demuxPrep.PrepareDemuxI(fwi.id, fwi.rxl.InterestDemux())
	demuxPrep.PrepareDemuxD(fwi.rxl.DataDemux())
	demuxPrep.PrepareDemuxN(fwi.rxl.NackDemux())

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
