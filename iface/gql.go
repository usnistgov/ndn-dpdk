package iface

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/core/nnduration"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlPktQueueInput *graphql.InputObject
	GqlCountersType  *graphql.Object
	GqlFaceNodeType  *gqlserver.NodeType
	GqlFaceType      *graphql.Object
)

func init() {
	GqlPktQueueInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FacePktQueueInput",
		Description: "Packet queue configuration.",
		Fields: graphql.InputObjectConfigFieldMap{
			"capacity": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"dequeueBurstSize": &graphql.InputObjectFieldConfig{
				Type: graphql.Int,
			},
			"delay": &graphql.InputObjectFieldConfig{
				Type: nnduration.GqlNanoseconds,
			},
			"disableCoDel": &graphql.InputObjectFieldConfig{
				Type: graphql.Boolean,
			},
			"target": &graphql.InputObjectFieldConfig{
				Type: nnduration.GqlNanoseconds,
			},
			"interval": &graphql.InputObjectFieldConfig{
				Type: nnduration.GqlNanoseconds,
			},
		},
	})

	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FaceCounters",
		Fields: gqlserver.BindFields(Counters{}, nil),
	})

	GqlFaceNodeType = gqlserver.NewNodeType((*Face)(nil))
	GqlFaceNodeType.Retrieve = func(id string) (interface{}, error) {
		nid, e := strconv.Atoi(id)
		if e != nil {
			return nil, e
		}
		return Get(ID(nid)), nil
	}
	GqlFaceNodeType.Delete = func(source interface{}) error {
		face := source.(Face)
		return face.Close()
	}

	GqlFaceType = graphql.NewObject(GqlFaceNodeType.Annotate(graphql.ObjectConfig{
		Name: "Face",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Type:        gqlserver.NonNullInt,
				Description: "Numeric face identifier.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					face := p.Source.(Face)
					return int(face.ID()), nil
				},
			},
			"locator": &graphql.Field{
				Type:        gqlserver.NonNullJSON,
				Description: "Endpoint addresses.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					face := p.Source.(Face)
					locw := LocatorWrapper{
						Locator: face.Locator(),
					}
					return locw, nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
			"counters": &graphql.Field{
				Type:        GqlCountersType,
				Description: "Face counters.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					face := p.Source.(Face)
					return face.Counters(), nil
				},
			},
		},
	}))
	GqlFaceNodeType.Register(GqlFaceType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "faces",
		Description: "List of faces.",
		Type:        gqlserver.NewNonNullList(GqlFaceType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
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
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var locw LocatorWrapper
			if e := jsonhelper.Roundtrip(p.Args["locator"], &locw, jsonhelper.DisallowUnknownFields); e != nil {
				return nil, e
			}
			return locw.Locator.CreateFace()
		},
	})
}
