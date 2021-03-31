package ethface

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/iface"
)

func init() {
	resolvePort := func(p graphql.ResolveParams) *Port {
		dev := p.Source.(ethdev.EthDev)
		return portByEthDev[dev]
	}
	ethdev.GqlEthDevType.AddFieldConfig("implName", &graphql.Field{
		Description: "Active ethface internal implementation name.",
		Type:        graphql.String,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			port := resolvePort(p)
			if port == nil {
				return nil, nil
			}
			return port.ImplName(), nil
		},
	})
	ethdev.GqlEthDevType.AddFieldConfig("faces", &graphql.Field{
		Description: "Faces on Ethernet device.",
		Type:        graphql.NewList(graphql.NewNonNull(iface.GqlFaceType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			port := resolvePort(p)
			if port == nil {
				return nil, nil
			}
			return port.Faces(), nil
		},
	})

	iface.GqlFaceType.AddFieldConfig("ethDev", &graphql.Field{
		Description: "Ethernet device containing this face.",
		Type:        ethdev.GqlEthDevType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			face := p.Source.(iface.Face)
			ethFace, ok := face.(*ethFace)
			if !ok {
				return nil, nil
			}
			return ethFace.port.dev, nil
		},
	})
}
