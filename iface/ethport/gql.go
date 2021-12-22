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
	ethdev.GqlEthDevType.AddFieldConfig("rxImpl", &graphql.Field{
		Description: "Active ethface RX implementation.",
		Type:        graphql.String,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			port := Find(p.Source.(ethdev.EthDev))
			if port == nil {
				return nil, nil
			}
			return port.rxImpl.String(), nil
		},
	})
	ethdev.GqlEthDevType.AddFieldConfig("faces", &graphql.Field{
		Description: "Faces on Ethernet device.",
		Type:        graphql.NewList(graphql.NewNonNull(iface.GqlFaceType)),
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
			port := Find(p.Source.(ethdev.EthDev))
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
		Args:        gqlserver.BindArguments(Config{}, ethnetif.GqlConfigFieldTypes),
		Type:        ethdev.GqlEthDevType,
		Resolve: func(p graphql.ResolveParams) (interface{}, error) {
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

	ethdev.GqlEthDevNodeType.Delete = func(source interface{}) error {
		dev := source.(ethdev.EthDev)
		port := Find(dev)
		if port == nil {
			return dev.Close()
		}
		return port.Close()
	}
}
