package iface

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func init() {
	ocFace := graphql.ObjectConfig{
		Name: "Face",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Type:        graphql.NewNonNull(graphql.Int),
				Description: "Numeric face identifier.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					face := p.Source.(Face)
					return int(face.ID()), nil
				},
			},
			"locator": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "Endpoint addresses.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					face := p.Source.(Face)
					return face.Locator(), nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
			"counters": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "Face counters.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					face := p.Source.(Face)
					return face.ReadCounters(), nil
				},
			},
		},
	}

	ntFace := gqlserver.NodeType("face")
	ntFace.Annotate(&ocFace, func(source interface{}) string {
		face := source.(Face)
		return strconv.Itoa(int(face.ID()))
	})
	tFace := graphql.NewObject(ocFace)
	ntFace.Register(tFace, (*Face)(nil), func(id string) (interface{}, error) {
		nid, e := strconv.Atoi(id)
		if e != nil {
			return nil, e
		}
		return Get(ID(nid)), nil
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "faces",
		Description: "List of faces.",
		Type:        graphql.NewList(tFace),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return List(), nil
		},
	})
}
