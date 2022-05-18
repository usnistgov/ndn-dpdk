package ethport

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
	GqlRxGroupInterface *gqlserver.Interface
	GqlRxgFlowType      *graphql.Object
	GqlRxgTableType     *graphql.Object
)

func gqlDefineRxGroup[T iface.RxGroup](oc graphql.ObjectConfig) *graphql.Object {
	iface.GqlRxGroupInterface.AppendTo(&oc)
	GqlRxGroupInterface.AppendTo(&oc)
	oc.Fields = GqlRxGroupInterface.CopyFieldsTo(oc.Fields)
	obj := graphql.NewObject(oc)
	gqlserver.ImplementsInterface[T](obj, iface.GqlRxGroupInterface)
	gqlserver.ImplementsInterface[T](obj, GqlRxGroupInterface)
	return obj
}

func init() {
	GqlRxGroupInterface = gqlserver.NewInterface(graphql.InterfaceConfig{
		Name: "EthRxGroup",
		Fields: iface.GqlRxGroupInterface.CopyFieldsTo(graphql.Fields{
			"port": &graphql.Field{
				Type: graphql.NewNonNull(ethdev.GqlEthDevType.Object),
			},
			"queue": &graphql.Field{
				Type: gqlserver.NonNullInt,
			},
		}),
	})

	GqlRxgFlowType = gqlDefineRxGroup[*rxgFlow](graphql.ObjectConfig{
		Name: "EthRxgFlow",
		Fields: graphql.Fields{
			"port": &graphql.Field{
				Resolve: func(p graphql.ResolveParams) (any, error) {
					rxf := p.Source.(*rxgFlow)
					return rxf.face.port.EthDev(), nil
				},
			},
			"queue": &graphql.Field{
				Resolve: func(p graphql.ResolveParams) (any, error) {
					rxf := p.Source.(*rxgFlow)
					return rxf.queue, nil
				},
			},
		},
	})

	GqlRxgTableType = gqlDefineRxGroup[*rxgTable](graphql.ObjectConfig{
		Name: "EthRxgTable",
		Fields: graphql.Fields{
			"port": &graphql.Field{
				Resolve: func(p graphql.ResolveParams) (any, error) {
					rxt := p.Source.(*rxgTable)
					return rxt.ethDev(), nil
				},
			},
			"queue": &graphql.Field{
				Resolve: func(p graphql.ResolveParams) (any, error) {
					rxt := p.Source.(*rxgTable)
					return int(rxt.queue), nil
				},
			},
		},
	})

	ethdev.GqlEthDevType.Object.AddFieldConfig("rxGroups", &graphql.Field{
		Description: "RxGroups on Ethernet device.",
		Type:        gqlserver.NewListNonNullElem(GqlRxGroupInterface.Interface),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			port := Find(p.Source.(ethdev.EthDev))
			if port == nil {
				return nil, nil
			}
			return port.rxImpl.List(port), nil
		},
	})
	ethdev.GqlEthDevType.Object.AddFieldConfig("faces", &graphql.Field{
		Description: "Faces on Ethernet device.",
		Type:        gqlserver.NewListNonNullElem(iface.GqlFaceType.Object),
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
