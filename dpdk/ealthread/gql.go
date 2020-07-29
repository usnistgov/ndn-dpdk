package ealthread

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

func init() {
	ntWorker := gqlserver.NewNodeType(eal.LCore{})
	tWorker := graphql.NewObject(ntWorker.Annotate(graphql.ObjectConfig{
		Name: "Worker",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Type:        gqlserver.NonNullInt,
				Description: "Numeric LCore ID.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					lc := p.Source.(eal.LCore)
					return lc.ID(), nil
				},
			},
			"isBusy": &graphql.Field{
				Type:        gqlserver.NonNullBoolean,
				Description: "Whether the LCore is running",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					lc := p.Source.(eal.LCore)
					return lc.IsBusy(), nil
				},
			},
			"role": &graphql.Field{
				Type:        graphql.String,
				Description: "Assigned role.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					lc := p.Source.(eal.LCore)
					return gqlserver.Optional(DefaultAllocator.allocated[lc.ID()]), nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
		},
	}))
	ntWorker.Retrieve = func(id string) (interface{}, error) {
		nid, e := strconv.Atoi(id)
		if e != nil {
			return nil, e
		}
		for _, lc := range DefaultAllocator.provider.Workers() {
			if lc.ID() == nid {
				return lc, nil
			}
		}
		return nil, nil
	}
	ntWorker.Register(tWorker)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "workers",
		Description: "Worker LCore allocations.",
		Type:        graphql.NewList(tWorker),
		Args: graphql.FieldConfigArgument{
			"role": &graphql.ArgumentConfig{
				Type:        graphql.String,
				Description: "Filter by assigned role. Empty string matches unassigned LCores.",
			},
			"numaSocket": &graphql.ArgumentConfig{
				Type:        graphql.Int,
				Description: "Filter by NUMA socket.",
			},
		},
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			roleFilter := func(lc eal.LCore) bool { return true }
			if role, ok := p.Args["role"].(string); ok {
				roleFilter = func(lc eal.LCore) bool { return DefaultAllocator.allocated[lc.ID()] == role }
			}
			numaSocketFilter := func(lc eal.LCore) bool { return true }
			if numaSocket, ok := p.Args["numaSocket"].(int); ok {
				numaSocketFilter = func(lc eal.LCore) bool { return DefaultAllocator.provider.NumaSocketOf(lc).ID() == numaSocket }
			}

			var list []eal.LCore
			for _, lc := range DefaultAllocator.provider.Workers() {
				if roleFilter(lc) && numaSocketFilter(lc) {
					list = append(list, lc)
				}
			}
			return list, nil
		},
	})
}
