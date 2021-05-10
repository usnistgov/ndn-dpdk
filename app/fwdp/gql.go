package fwdp

/*
#include "../../csrc/fwdp/fwd.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"strconv"
	"unsafe"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/container/cs/cscnt"
	"github.com/usnistgov/ndn-dpdk/container/pit"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/runningstat"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
	"github.com/usnistgov/ndn-dpdk/iface"
)

var (
	// GqlDataPlane is the DataPlane instance accessible via GraphQL.
	GqlDataPlane *DataPlane

	errNoGqlDataPlane = errors.New("DataPlane unavailable")
)

// GraphQL types.
var (
	GqlInputNodeType   *gqlserver.NodeType
	GqlInputType       *graphql.Object
	GqlFwdCountersType *graphql.Object
	GqlFwdNodeType     *gqlserver.NodeType
	GqlFwdType         *graphql.Object
	GqlDataPlaneType   *graphql.Object
)

func init() {
	GqlInputNodeType = gqlserver.NewNodeType((*Input)(nil))
	GqlInputNodeType.Retrieve = func(id string) (interface{}, error) {
		if GqlDataPlane == nil {
			return nil, errNoGqlDataPlane
		}
		i, e := strconv.Atoi(id)
		if e != nil || i < 0 || i >= len(GqlDataPlane.inputs) {
			return nil, nil
		}
		return GqlDataPlane.inputs[i], nil
	}

	GqlInputType = graphql.NewObject(GqlInputNodeType.Annotate(graphql.ObjectConfig{
		Name: "FwInput",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Input thread index.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					input := p.Source.(*Input)
					return input.id, nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
		},
	}))
	GqlInputNodeType.Register(GqlInputType)

	GqlFwdCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "FwFwdCounters",
		Fields: graphql.Fields{
			"inputLatency": &graphql.Field{
				Description: "Latency between packet arrival and dequeuing at forwarding thread, in nanoseconds.",
				Type:        graphql.NewNonNull(runningstat.GqlSnapshotType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					latencyStat := runningstat.FromPtr(unsafe.Pointer(&fwd.c.latencyStat))
					return latencyStat.Read().Scale(eal.GetNanosInTscUnit()), nil
				},
			},
			"nNoFibMatch": &graphql.Field{
				Description: "Interests dropped due to no FIB match.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return int(fwd.c.nNoFibMatch), nil
				},
			},
			"nDupNonce": &graphql.Field{
				Description: "Interests dropped due to duplicate nonce.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return int(fwd.c.nDupNonce), nil
				},
			},
			"nSgNoFwd": &graphql.Field{
				Description: "Interests not forwarded by strategy.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return int(fwd.c.nSgNoFwd), nil
				},
			},
			"nNackMismatch": &graphql.Field{
				Description: "Nacks dropped due to outdated nonce.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return int(fwd.c.nNackMismatch), nil
				},
			},
		},
	})
	defineFwdPktCounter := func(plural string, getDemux func(iface.RxLoop) *iface.InputDemux, getQueue func(fwdC *C.FwFwd) *C.PktQueue) {
		GqlFwdCountersType.AddFieldConfig(fmt.Sprintf("n%sQueued", plural), &graphql.Field{
			Description: fmt.Sprintf("%s queued in input thread.", plural),
			Type:        gqlserver.NonNullInt,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				index := p.Source.(*Fwd).id
				var sum uint64
				for _, input := range GqlDataPlane.inputs {
					sum += getDemux(input.rxl).ReadDestCounters(index).NQueued
				}
				return sum, nil
			},
		})
		GqlFwdCountersType.AddFieldConfig(fmt.Sprintf("n%sDropped", plural), &graphql.Field{
			Description: fmt.Sprintf("%s dropped in input thread.", plural),
			Type:        gqlserver.NonNullInt,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				index := p.Source.(*Fwd).id
				var sum uint64
				for _, input := range GqlDataPlane.inputs {
					sum += getDemux(input.rxl).ReadDestCounters(index).NDropped
				}
				return sum, nil
			},
		})
		GqlFwdCountersType.AddFieldConfig(fmt.Sprintf("n%sCongMarked", plural), &graphql.Field{
			Description: fmt.Sprintf("Congestion marks added to %s.", plural),
			Type:        gqlserver.NonNullInt,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				fwd := p.Source.(*Fwd)
				return int(getQueue(fwd.c).nDrops), nil
			},
		})
	}
	defineFwdPktCounter("Interests", iface.RxLoop.InterestDemux, func(fwdC *C.FwFwd) *C.PktQueue { return &fwdC.queueI })
	defineFwdPktCounter("Data", iface.RxLoop.DataDemux, func(fwdC *C.FwFwd) *C.PktQueue { return &fwdC.queueD })
	defineFwdPktCounter("Nacks", iface.RxLoop.NackDemux, func(fwdC *C.FwFwd) *C.PktQueue { return &fwdC.queueN })

	GqlFwdNodeType = gqlserver.NewNodeType((*Fwd)(nil))
	GqlFwdNodeType.Retrieve = func(id string) (interface{}, error) {
		if GqlDataPlane == nil {
			return nil, errNoGqlDataPlane
		}
		i, e := strconv.Atoi(id)
		if e != nil || i < 0 || i >= len(GqlDataPlane.fwds) {
			return nil, nil
		}
		return GqlDataPlane.fwds[i], nil
	}

	GqlFwdType = graphql.NewObject(GqlFwdNodeType.Annotate(graphql.ObjectConfig{
		Name: "FwFwd",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Description: "Forwarding thread index.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return fwd.id, nil
				},
			},
			"worker": ealthread.GqlWithWorker(nil),
			"counters": &graphql.Field{
				Description: "Forwarding counters.",
				Type:        graphql.NewNonNull(GqlFwdCountersType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source, nil
				},
			},
			"pitCounters": &graphql.Field{
				Description: "PIT counters.",
				Type:        graphql.NewNonNull(pit.GqlCountersType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return fwd.Pit().Counters(), nil
				},
			},
			"csCounters": &graphql.Field{
				Description: "CS counters.",
				Type:        graphql.NewNonNull(cscnt.GqlCountersType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					fwd := p.Source.(*Fwd)
					return cscnt.ReadCounters(fwd.Pit(), fwd.Cs()), nil
				},
			},
		},
	}))
	GqlFwdNodeType.Register(GqlFwdType)

	GqlDataPlaneType = graphql.NewObject(graphql.ObjectConfig{
		Name: "FwDataPlane",
		Fields: graphql.Fields{
			"inputs": &graphql.Field{
				Description: "Input threads.",
				Type:        gqlserver.NewNonNullList(GqlInputType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					dp := p.Source.(*DataPlane)
					return dp.inputs, nil
				},
			},
			"fwds": &graphql.Field{
				Description: "Forwarding threads.",
				Type:        gqlserver.NewNonNullList(GqlFwdType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					dp := p.Source.(*DataPlane)
					return dp.fwds, nil
				},
			},
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "fwdp",
		Description: "Forwarder data plane.",
		Type:        GqlDataPlaneType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return GqlDataPlane, nil
		},
	})
}
