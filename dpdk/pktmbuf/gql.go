package pktmbuf

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlConfigType   *graphql.Object
	GqlPoolType     *graphql.Object
	GqlTemplateType *graphql.Object
)

func init() {
	GqlConfigType = graphql.NewObject(graphql.ObjectConfig{
		Name:   "PktmbufPoolConfig",
		Fields: gqlserver.BindFields(PoolConfig{}, nil),
	})

	GqlPoolType = graphql.NewObject(graphql.ObjectConfig{
		Name: "PktmbufPool",
		Fields: graphql.Fields{
			"name": &graphql.Field{
				Description: "Mempool name.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					pool := p.Source.(PoolInfo)
					return pool.String(), nil
				},
			},
			"used": &graphql.Field{
				Description: "Entries in use.",
				Type:        gqlserver.NonNullInt,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					pool := p.Source.(PoolInfo)
					return pool.CountInUse(), nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
		},
	})

	GqlTemplateType = graphql.NewObject(graphql.ObjectConfig{
		Name: "PktmbufPoolTemplate",
		Fields: graphql.Fields{
			"tid": &graphql.Field{
				Description: "Template ID.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					tpl := p.Source.(Template)
					return tpl.ID(), nil
				},
			},
			"config": &graphql.Field{
				Description: "Mempool configuration.",
				Type:        graphql.NewNonNull(GqlConfigType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					tpl := p.Source.(Template)
					return tpl.Config(), nil
				},
			},
			"pools": &graphql.Field{
				Description: "List of created mempools.",
				Type:        gqlserver.NewNonNullList(GqlPoolType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					tpl := p.Source.(Template)
					return tpl.Pools(), nil
				},
			},
		},
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "pktmbufPoolTemplates",
		Description: "Packet buffer pool templates.",
		Type:        gqlserver.NewNonNullList(GqlTemplateType),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var list []Template
			for _, tpl := range templates {
				list = append(list, tpl)
			}
			return list, nil
		},
	})
}
