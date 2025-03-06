package ethdev

import (
	"errors"
	"fmt"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlEthDevType *gqlserver.NodeType[EthDev]
)

func init() {
	GqlEthDevType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name: "EthDev",
		Fields: graphql.Fields{
			"nid": &graphql.Field{
				Type:        gqlserver.NonNullInt,
				Description: "DPDK port identifier.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					return port.ID(), nil
				},
			},
			"name": &graphql.Field{
				Type:        gqlserver.NonNullString,
				Description: "Port name.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					return port.Name(), nil
				},
			},
			"numaSocket": eal.GqlWithNumaSocket,
			"devInfo": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "DPDK device information.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					return port.DevInfo(), nil
				},
			},
			"rxqInfo": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "DPDK RX queues information.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					queues := port.RxQueues()
					list := make([]RxqInfo, len(queues))
					for i, q := range queues {
						list[i] = q.Info()
					}
					return list, nil
				},
			},
			"txqInfo": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "DPDK TX queues information.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					queues := port.TxQueues()
					list := make([]TxqInfo, len(queues))
					for i, q := range queues {
						list[i] = q.Info()
					}
					return list, nil
				},
			},
			"macAddr": &graphql.Field{
				Type:        gqlserver.NonNullString,
				Description: "MAC address.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					return port.HardwareAddr(), nil
				},
			},
			"mtu": &graphql.Field{
				Type:        gqlserver.NonNullInt,
				Description: "Maximum Transmission Unit (MTU).",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					return port.MTU(), nil
				},
			},
			"isDown": &graphql.Field{
				Type:        gqlserver.NonNullBoolean,
				Description: "Whether the port is down.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					return port.IsDown(), nil
				},
			},
			"stats": &graphql.Field{
				Type:        gqlserver.JSON,
				Description: "Hardware statistics.",
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					return port.Stats(), nil
				},
			},
			"flowDump": &graphql.Field{
				Description: "Internal rte_flow representation.",
				Type:        gqlserver.NonNullString,
				Resolve: func(p graphql.ResolveParams) (any, error) {
					port := p.Source.(EthDev)
					dump, e := GetFlowDump(port)
					if e != nil {
						return fmt.Sprint("ERROR: ", e), nil
					}
					return string(dump), nil
				},
			},
		},
	}, gqlserver.NodeConfig[EthDev]{
		RetrieveInt: FromID,
	})

	gqlserver.AddQuery(&graphql.Field{
		Name:        "ethDevs",
		Description: "List of Ethernet devices.",
		Type:        gqlserver.NewListNonNullBoth(GqlEthDevType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
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
		Type: graphql.NewNonNull(GqlEthDevType.Object),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			port := GqlEthDevType.Retrieve(p.Args["id"].(string))
			if port == nil {
				return nil, errors.New("port not found")
			}
			port.ResetStats()
			return port, nil
		},
	})
}
