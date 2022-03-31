package ethdev

/*
#include "../../csrc/dpdk/ethdev.h"
*/
import "C"
import (
	"fmt"
	"strconv"
	"unsafe"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/cptr"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/dpdk/eal"
)

// GraphQL types.
var (
	GqlEthDevNodeType *gqlserver.NodeType
	GqlEthDevType     *graphql.Object
)

func init() {
	GqlEthDevNodeType = gqlserver.NewNodeType((*EthDev)(nil))
	GqlEthDevNodeType.Retrieve = func(id string) (any, error) {
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
					port := p.Source.(ethDev)
					var res C.int
					dump, e := cptr.CaptureFileDump(func(fp unsafe.Pointer) {
						res = C.rte_flow_dev_dump(port.cID(), nil, (*C.FILE)(fp), nil)
					})
					if e == nil && res != 0 {
						return fmt.Sprint("ERROR: ", eal.MakeErrno(res)), nil
					}
					return string(dump), e
				},
			},
		},
	}))
	GqlEthDevNodeType.Register(GqlEthDevType)

	gqlserver.AddQuery(&graphql.Field{
		Name:        "ethDevs",
		Description: "List of Ethernet devices.",
		Type:        gqlserver.NewNonNullList(GqlEthDevType),
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
		Type: graphql.NewNonNull(GqlEthDevType),
		Resolve: func(p graphql.ResolveParams) (any, error) {
			var port EthDev
			if e := gqlserver.RetrieveNodeOfType(GqlEthDevNodeType, p.Args["id"], &port); e != nil {
				return nil, e
			}
			port.ResetStats()
			return port, nil
		},
	})
}
