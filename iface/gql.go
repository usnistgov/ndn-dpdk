package iface

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlCountersType *graphql.Object
	GqlFaceNodeType *gqlserver.NodeType
	GqlFaceType     *graphql.Object
)

func init() {
	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "FaceCounters",
		Fields: graphql.BindFields(Counters{}),
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
					return face.Locator(), nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
			"counters": &graphql.Field{
				Type:        GqlCountersType,
				Description: "Face counters.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					face := p.Source.(Face)
					return face.ReadCounters(), nil
				},
			},
		},
	}))
	GqlFaceNodeType.Register(GqlFaceType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "faces",
		Description: "List of faces.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlFaceType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return List(), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createFace",
		Description: "Create a face.",
		Args: graphql.FieldConfigArgument{
			"locator": &graphql.ArgumentConfig{
				Type: gqlserver.NonNullJSON,
			},
		},
		Type: GqlFaceType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var locw LocatorWrapper
			if e := gqlserver.DecodeJSON(p.Args["locator"], &locw); e != nil {
				return nil, e
			}
			return locw.Locator.CreateFace()
		},
	})
}
