package fwdp

/*
#include "../../csrc/fwdp/token.h"
*/
import "C"
import (
	"github.com/usnistgov/ndn-dpdk/container/ndt"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndni"
)

// DispatchThread represents a thread that dispatches packets to forwarding threads.
// It could be an input, crypto, or disk service thread.
type DispatchThread interface {
	// DispatchThreadID returns numeric index of the dispatch thread.
	// IDs should be sequentially assigned.
	DispatchThreadID() int

	// WithInputDemuxes contains DemuxOf function that returns InputDemux.
	// If the dispatch thread does not handle a particular packet type, that function returns nil.
	iface.WithInputDemuxes
}

// DispatchCounters contains counters of packets dispatched from a thread toward forwarding threads.
type DispatchCounters struct {
	NInterestsQueued  []uint64 `json:"nInterestsQueued" gqldesc:"Interests enqueued toward each forwarding thread."`
	NInterestsDropped []uint64 `json:"nInterestsDropped" gqldesc:"Interests dropped toward each forwarding thread."`
	NDataQueued       []uint64 `json:"nDataQueued" gqldesc:"Data enqueued toward each forwarding thread."`
	NDataDropped      []uint64 `json:"nDataDropped" gqldesc:"Data dropped toward each forwarding thread."`
	NNacksQueued      []uint64 `json:"nNacksQueued" gqldesc:"Nacks enqueued toward each forwarding thread."`
	NNacksDropped     []uint64 `json:"nNacksDropped" gqldesc:"Nacks dropped toward each forwarding thread."`
}

// ReadDispatchCounters retrieves DispatchCounters.
func ReadDispatchCounters(th DispatchThread, nFwds int) (cnt DispatchCounters) {
	for _, t := range []struct {
		PktType ndni.PktType
		Queued  *[]uint64
		Dropped *[]uint64
	}{
		{ndni.PktInterest, &cnt.NInterestsQueued, &cnt.NInterestsDropped},
		{ndni.PktData, &cnt.NDataQueued, &cnt.NDataDropped},
		{ndni.PktNack, &cnt.NNacksQueued, &cnt.NNacksDropped},
	} {
		demux := th.DemuxOf(t.PktType)
		if demux == nil {
			continue
		}
		for i := range nFwds {
			dest := demux.DestCounters(i)
			*t.Queued = append(*t.Queued, dest.NQueued)
			*t.Dropped = append(*t.Dropped, dest.NDropped)
		}
	}
	return
}

// demuxPreparer contains contextual information for preparing InputDemux of each packet type.
type demuxPreparer struct {
	Ndt  *ndt.Ndt
	Fwds []*Fwd
}

func (p *demuxPreparer) Prepare(th DispatchThread, socket eal.NumaSocket) {
	if demuxI := th.DemuxOf(ndni.PktInterest); demuxI != nil {
		p.PrepareDemuxI(demuxI, socket)
	}
	if demuxD := th.DemuxOf(ndni.PktData); demuxD != nil {
		p.PrepareDemuxD(demuxD)
	}
	if demuxN := th.DemuxOf(ndni.PktNack); demuxN != nil {
		p.PrepareDemuxN(demuxN)
	}
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
