package ethface

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// GraphQL types.
var (
	GqlRxImplKind *graphql.Enum
)

func init() {
	GqlRxImplKind = gqlserver.NewStringEnum("EthFaceRxImplKind", "Port RX implementation.", RxImplMemif, RxImplFlow, RxImplTable)

	ethdev.GqlEthDevType.AddFieldConfig("rxImpl", &graphql.Field{
		Description: "Active ethface RX implementation.",
		Type:        GqlRxImplKind,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			port := FindPort(p.Source.(ethdev.EthDev))
			if port == nil {
				return nil, nil
			}
			return port.rxImpl.Kind(), nil
		},
	})
	ethdev.GqlEthDevType.AddFieldConfig("faces", &graphql.Field{
		Description: "Faces on Ethernet device.",
		Type:        graphql.NewList(graphql.NewNonNull(iface.GqlFaceType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			port := FindPort(p.Source.(ethdev.EthDev))
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

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createEthPort",
		Description: "Create an Ethernet port.",
		Args:        gqlserver.BindArguments(PortConfig{}, ethnetif.GqlConfigFieldTypes),
		Type:        ethdev.GqlEthDevType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			var cfg PortConfig
			if e := jsonhelper.Roundtrip(p.Args, &cfg); e != nil {
				return nil, e
			}

			port, e := NewPort(cfg)
			if e != nil {
				return nil, e
			}
			return port.dev, nil
		},
	})
}
