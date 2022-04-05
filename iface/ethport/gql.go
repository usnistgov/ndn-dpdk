package ethport

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/core/jsonhelper"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev"
	"github.com/usnistgov/ndn-dpdk/dpdk/ethdev/ethnetif"
	"github.com/usnistgov/ndn-dpdk/iface"
)

func init() {
	ethdev.GqlEthDevType.Object.AddFieldConfig("rxImpl", &graphql.Field{
		Description: "Active ethface RX implementation.",
		Type:        graphql.String,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			port := Find(p.Source.(ethdev.EthDev))
			if port == nil {
				return nil, nil
			}
			return port.rxImpl.String(), nil
		},
	})
	ethdev.GqlEthDevType.Object.AddFieldConfig("faces", &graphql.Field{
		Description: "Faces on Ethernet device.",
		Type:        graphql.NewList(graphql.NewNonNull(iface.GqlFaceType.Object)),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			port := Find(p.Source.(ethdev.EthDev))
			if port == nil {
				return nil, nil
			}
			return port.Faces(), nil
		},
	})

	iface.GqlFaceType.Object.AddFieldConfig("ethDev", &graphql.Field{
		Description: "Ethernet device containing this face.",
		Type:        ethdev.GqlEthDevType.Object,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			face, ok := p.Source.(*Face)
			if !ok {
				return nil, nil
			}
			return face.port.dev, nil
		},
	})

	gqlserver.AddMutation(&graphql.Field{
		Name:        "createEthPort",
		Description: "Create an Ethernet port.",
		Args:        gqlserver.BindArguments[Config](ethnetif.GqlConfigFieldTypes),
		Type:        ethdev.GqlEthDevType.Object,
		Resolve: func(p graphql.ResolveParams) (any, error) {
			var cfg Config
			if e := jsonhelper.Roundtrip(p.Args, &cfg); e != nil {
				return nil, e
			}

			port, e := New(cfg)
			if e != nil {
				return nil, e
			}
			return port.dev, nil
		},
	})
}
