package socketface

import (
	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/iface"
)

// GraphQL types.
var (
	GqlRxConnsType *graphql.Object
	GqlRxEpollType *graphql.Object
)

func init() {
	ocRxConns := graphql.ObjectConfig{
		Name:   "SocketRxConns",
		Fields: iface.GqlRxGroupInterface.CopyFieldsTo(nil),
	}
	iface.GqlRxGroupInterface.AppendTo(&ocRxConns)
	GqlRxConnsType = graphql.NewObject(ocRxConns)
	gqlserver.ImplementsInterface[*rxConns](GqlRxConnsType, iface.GqlRxGroupInterface)

	ocRxEpoll := graphql.ObjectConfig{
		Name:   "SocketRxEpoll",
		Fields: iface.GqlRxGroupInterface.CopyFieldsTo(nil),
	}
	iface.GqlRxGroupInterface.AppendTo(&ocRxEpoll)
	GqlRxEpollType = graphql.NewObject(ocRxEpoll)
	gqlserver.ImplementsInterface[*rxEpoll](GqlRxEpollType, iface.GqlRxGroupInterface)
}
