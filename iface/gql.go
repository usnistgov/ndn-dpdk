package iface

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
	"github.com/usnistgov/ndn-dpdk/dpdk/ealthread"
)

var (
	// GqlCreateFaceAllowed indicates whether face creation via GraphQL is allowed.
	GqlCreateFaceAllowed bool

	errGqlCreateFaceDisallowed = errors.New("createFace is disallowed; is NDN-DPDK forwarder activated?")
)

// GraphQL types.
var (
	GqlPktQueueInput  *graphql.InputObject
	GqlFaceNodeType   *gqlserver.NodeType
	GqlFaceType       *graphql.Object
	GqlRxCountersType *graphql.Object
	GqlTxCountersType *graphql.Object
	GqlCountersType   *graphql.Object
)

func init() {
	GqlPktQueueInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FacePktQueueInput",
		Description: "Packet queue configuration.",
		Fields: gqlserver.BindInputFields(PktQueueConfig{}, gqlserver.FieldTypes{
			reflect.TypeOf(nnduration.Nanoseconds(0)): nnduration.GqlNanoseconds,
		}),
	})

	GqlFaceNodeType = gqlserver.NewNodeType((*Face)(nil))
	GqlFaceNodeType.Retrieve = func(id string) (any, error) {
		nid, e := strconv.Atoi(id)
		if e != nil {
			return nil, e
		}
		return Get(ID(nid)), nil
	}
	GqlFaceNodeType.Delete = func(source any) error {
		face := source.(Face)
		return face.Close()
	}

	GqlFaceType = graphql.NewObject(GqlFaceNodeType.Annotate(graphql.ObjectConfig{
		Name: "Face",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Type:        gqlserver.NonNullInt,
				Description: "Numeric face identifier.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					face := p.Source.(Face)
					return int(face.ID()), nil
				},
			},
			"locator": &graphql.Field{
				Type:        gqlserver.NonNullJSON,
				Description: "Endpoint addresses.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					face := p.Source.(Face)
					locw := LocatorWrapper{
						Locator: face.Locator(),
					}
					return locw, nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
			"txLoop": &graphql.Field{
				Type:        ealthread.GqlWorkerType,
				Description: "TxLoop serving this face.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					face := p.Source.(Face)
					txl := mapFaceTxl[face.ID()]
					if txl == nil {
						return nil, nil
					}
					lc := txl.LCore()
					return gqlserver.Optional(lc, lc.Valid()), nil
				},
			},
		},
	}))
	GqlFaceNodeType.Register(GqlFaceType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "faces",
		Description: "List of faces.",
		Type:        gqlserver.NewNonNullList(GqlFaceType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			return List(), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createFace",
		Description: "Create a face.",
		Args: graphql.FieldConfigArgument{
			"locator": &graphql.ArgumentConfig{
				Description: "JSON object that satisfies the schema given in 'locator.schema.json'.",
				Type:        gqlserver.NonNullJSON,
			},
		},
		Type: graphql.NewNonNull(GqlFaceType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			if !GqlCreateFaceAllowed {
				return nil, errGqlCreateFaceDisallowed
			}

			var locw LocatorWrapper
			if e := jsonhelper.Roundtrip(p.Args["locator"], &locw, jsonhelper.DisallowUnknownFields); e != nil {
				return nil, e
			}
			return locw.Locator.CreateFace()
		},
	})

	GqlRxCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FaceRxCounters",
		Fields: gqlserver.BindFields(RxCounters{}, nil),
	})
	GqlTxCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FaceTxCounters",
		Fields: gqlserver.BindFields(TxCounters{}, nil),
	})
	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "FaceCounters",
		Fields: gqlserver.BindFields(Counters{}, gqlserver.FieldTypes{
			reflect.TypeOf(RxCounters{}): GqlRxCountersType,
			reflect.TypeOf(TxCounters{}): GqlTxCountersType,
		}),
	})
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Face counters.",
		Parent:       GqlFaceType,
		Name:         "counters",
		Subscription: "faceCounters",
		FindArgs: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Face ID.",
				Type:        gqlserver.NonNullID,
			},
		},
		Find: func(p graphql.ResolveParams) (root any, enders []any, e error) {
			id := p.Args["id"].(string)
			var face Face
			if e := gqlserver.RetrieveNodeOfType(GqlFaceNodeType, id, &face); e != nil {
				return nil, nil, e
			}
			return face, nil, nil
		},
		Type: GqlCountersType,
		Read: func(p graphql.ResolveParams) (any, error) {
			face := p.Source.(Face)
			return face.Counters(), nil
		},
	})
}
