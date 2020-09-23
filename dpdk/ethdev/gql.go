package ethdev

import (
	"strconv"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlEthDevNodeType *gqlserver.NodeType
	GqlEthDevType     *graphql.Object
)

func init() {
	GqlEthDevNodeType = gqlserver.NewNodeType(EthDev{})
	GqlEthDevNodeType.Retrieve = func(id string) (interface{}, error) {
		nid, e := strconv.Atoi(id)
		if e != nil {
			return nil, e
		}
		return FromID(nid), nil
	}

	GqlEthDevType = graphql.NewObject(GqlEthDevNodeType.Annotate(graphql.ObjectConfig{
		Name: "EthDev",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Type:        gqlserver.NonNullInt,
				Description: "DPDK port identifier.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					port := p.Source.(EthDev)
					return port.ID(), nil
				},
			},
			"name": &graphql.Field{
				Type:        gqlserver.NonNullString,
				Description: "Port name.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					port := p.Source.(EthDev)
					return port.Name(), nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
			"devInfo": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "DPDK device information.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					port := p.Source.(EthDev)
					return port.DevInfo(), nil
				},
			},
			"macAddr": &graphql.Field{
				Type:        gqlserver.NonNullString,
				Description: "MAC address.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					port := p.Source.(EthDev)
					return port.MacAddr(), nil
				},
			},
			"mtu": &graphql.Field{
				Type:        gqlserver.NonNullInt,
				Description: "Maximum Transmission Unit (MTU).",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					port := p.Source.(EthDev)
					return port.MTU(), nil
				},
			},
			"isDown": &graphql.Field{
				Type:        gqlserver.NonNullBoolean,
				Description: "Whether the port is down.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					port := p.Source.(EthDev)
					return port.IsDown(), nil
				},
			},
			"stats": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "Hardware statistics.",
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					port := p.Source.(EthDev)
					return port.Stats(), nil
				},
			},
		},
	}))
	GqlEthDevNodeType.Register(GqlEthDevType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "ethDevs",
		Description: "List of Ethernet devices.",
		Type:        graphql.NewList(graphql.NewNonNull(GqlEthDevType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			return List(), nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "resetEthStats",
		Description: "Reset hardware statistics of an Ethernet device.",
		Args: graphql.FieldConfigArgument{
			"id": &graphql.ArgumentConfig{
				Type: gqlserver.NonNullID,
			},
		},
		Type: GqlEthDevType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			port, e := gqlserver.RetrieveNodeOfType(GqlEthDevNodeType, p.Args["id"])
			if e != nil {
				return nil, e
			}
			port.(EthDev).ResetStats()
			return port, nil
		},
	})
}
