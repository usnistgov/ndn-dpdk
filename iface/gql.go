package iface

import (
	"errors"
	"reflect"

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
	GqlPktQueueInput    *graphql.InputObject
	GqlFaceType         *gqlserver.NodeType[Face]
	GqlRxCountersType   *graphql.Object
	GqlTxCountersType   *graphql.Object
	GqlCountersType     *graphql.Object
	GqlRxGroupInterface *gqlserver.Interface
)

func init() {
	GqlPktQueueInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FacePktQueueInput",
		Description: "Packet queue configuration.",
		Fields: gqlserver.BindInputFields[PktQueueConfig](gqlserver.FieldTypes{
			reflect.TypeOf(nnduration.Nanoseconds(0)): nnduration.GqlNanoseconds,
		}),
	})

	GqlFaceType = gqlserver.NewNodeType(graphql.ObjectConfig{
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
			"isDown": &graphql.Field{
				Type:        gqlserver.NonNullBoolean,
				Description: "Whether the face is down.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					face := p.Source.(Face)
					return IsDown(face.ID()), nil
				},
			},
			"txLoop": &graphql.Field{
				Type:        ealthread.GqlWorkerType.Object,
				Description: "TxLoop serving this face.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					txLoopLock.Lock()
					defer txLoopLock.Unlock()
					face := p.Source.(Face)
					txl := mapFaceTxl[face.ID()]
					if txl == nil {
						return nil, nil
					}
					lc := txl.LCore()
					return gqlserver.Optional(lc), nil
				},
			},
		},
	}, gqlserver.NodeConfig[Face]{
		RetrieveInt: func(id int) Face {
			return Get(ID(id))
		},
		Delete: func(source Face) error {
			return source.Close()
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "faces",
		Description: "List of faces.",
		Type:        gqlserver.NewListNonNullBoth(GqlFaceType.Object),
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
		Type: graphql.NewNonNull(GqlFaceType.Object),
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
		Fields: gqlserver.BindFields[RxCounters](nil),
	})
	GqlTxCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FaceTxCounters",
		Fields: gqlserver.BindFields[TxCounters](nil),
	})
	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name: "FaceCounters",
		Fields: gqlserver.BindFields[Counters](gqlserver.FieldTypes{
			reflect.TypeOf(RxCounters{}): GqlRxCountersType,
			reflect.TypeOf(TxCounters{}): GqlTxCountersType,
		}),
	})
	gqlserver.AddCounters(&gqlserver.CountersConfig{
		Description:  "Face counters.",
		Parent:       GqlFaceType.Object,
		Name:         "counters",
		Subscription: "faceCounters",
		FindArgs: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Description: "Face ID.",
				Type:        gqlserver.NonNullID,
			},
		},
		Find: func(p graphql.ResolveParams) (root any, enders []any, e error) {
			face := GqlFaceType.Retrieve(p.Args["id"].(string))
			return face, nil, nil
		},
		Type: GqlCountersType,
		Read: func(p graphql.ResolveParams) (any, error) {
			face := p.Source.(Face)
			return face.Counters(), nil
		},
	})

	GqlRxGroupInterface = gqlserver.NewInterface(graphql.InterfaceConfig{
		Name: "RxGroup",
		Fields: graphql.Fields{
			"rxLoop": ealthread.GqlWithWorker(func(p graphql.ResolveParams) ealthread.Thread {
				rxLoopLock.Lock()
				defer rxLoopLock.Unlock()
				rxg := p.Source.(RxGroup)
				return mapRxgRxl[rxg]
			}),
			"faces": &graphql.Field{
				Type: gqlserver.NewListNonNullBoth(GqlFaceType.Object),
				Resolve: func(p graphql.ResolveParams) (any, error) {
					rxg := p.Source.(RxGroup)
					return rxg.Faces(), nil
				},
			},
		},
	})

	ealthread.GqlWorkerType.Object.AddFieldConfig("txLoopFaces", &graphql.Field{
		Description: "Faces on TX thread.",
		Type:        gqlserver.NewListNonNullElem(GqlFaceType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			txLoopLock.Lock()
			defer txLoopLock.Unlock()

			lc := p.Source.(eal.LCore)
			var txl TxLoop
			for t := range txLoopThreads {
				if t.LCore() == lc {
					txl = t
				}
			}
			if txl == nil {
				return nil, nil
			}

			list := []Face{}
			for id, t := range mapFaceTxl {
				if t == txl {
					list = append(list, Get(id))
				}
			}
			return list, nil
		},
	})
}

// RegisterGqlRxGroupType registers an implementation of GraphQL RxGroup interface.
func RegisterGqlRxGroupType[T RxGroup](oc graphql.ObjectConfig) (object *graphql.Object) {
	GqlRxGroupInterface.AppendTo(&oc)
	oc.Fields = GqlRxGroupInterface.CopyFieldsTo(oc.Fields)
	object = graphql.NewObject(oc)
	gqlserver.ImplementsInterface[T](object, GqlRxGroupInterface)
	return object
}
