package fileserver

import (
	"reflect"

	"github.com/graphql-go/graphql"
	"github.com/usnistgov/ndn-dpdk/app/tg/tggql"
	"github.com/usnistgov/ndn-dpdk/core/gqlserver"
	"github.com/usnistgov/ndn-dpdk/iface"
	"github.com/usnistgov/ndn-dpdk/ndn"
)

// GqlRetrieveByFaceID returns *FileServer associated with a face.
// It is assigned during package tg initialization.
var GqlRetrieveByFaceID func(id iface.ID) *Server

// GraphQL types.
var (
	GqlMountInput   *graphql.InputObject
	GqlConfigInput  *graphql.InputObject
	GqlCountersType *graphql.Object
	GqlServerType   *gqlserver.NodeType[*Server]
)

func init() {
	GqlMountInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FileServerMountInput",
		Description: "File server mount definition.",
		Fields: gqlserver.BindInputFields[Mount](gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}): gqlserver.NonNullString,
		}),
	})
	GqlConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FileServerConfigInput",
		Description: "File server config.",
		Fields: gqlserver.BindInputFields[Config](gqlserver.FieldTypes{
			reflect.TypeOf(iface.PktQueueConfig{}): iface.GqlPktQueueInput,
			reflect.TypeOf(Mount{}):                GqlMountInput,
		}),
	})
	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "FileServerCounters",
		Description: "File server counters.",
		Fields:      gqlserver.BindFields[Counters](nil),
	})

	GqlServerType = gqlserver.NewNodeType(graphql.ObjectConfig{
		Name:        "FileServer",
		Description: "File server.",
		Fields: tggql.CommonFields(graphql.Fields{
			"mounts": &graphql.Field{
				Description: "Mount entries.",
				Type:        gqlserver.NonNullJSON,
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*Server).Mounts(), nil
				},
			},
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        graphql.NewNonNull(GqlCountersType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*Server).Counters(), nil
				},
			},
		}),
	}, tggql.NodeConfig(&GqlRetrieveByFaceID))
}
