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
var GqlRetrieveByFaceID func(id iface.ID) any

// GraphQL types.
var (
	GqlMountInput     *graphql.InputObject
	GqlConfigInput    *graphql.InputObject
	GqlCountersType   *graphql.Object
	GqlServerNodeType *gqlserver.NodeType
	GqlServerType     *graphql.Object
)

func init() {
	GqlMountInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FileServerMountInput",
		Description: "File server mount definition.",
		Fields: gqlserver.BindInputFields(Mount{}, gqlserver.FieldTypes{
			reflect.TypeOf(ndn.Name{}): gqlserver.NonNullString,
		}),
	})
	GqlConfigInput = graphql.NewInputObject(graphql.InputObjectConfig{
		Name:        "FileServerConfigInput",
		Description: "File server config.",
		Fields: gqlserver.BindInputFields(Config{}, gqlserver.FieldTypes{
			reflect.TypeOf(iface.PktQueueConfig{}): iface.GqlPktQueueInput,
			reflect.TypeOf(Mount{}):                GqlMountInput,
		}),
	})
	GqlCountersType = graphql.NewObject(graphql.ObjectConfig{
		Name:        "FileServerCounters",
		Description: "File server counters.",
		Fields:      gqlserver.BindFields(Counters{}, nil),
	})

	GqlServerNodeType = tggql.NewNodeType("FileServer", (*Server)(nil), &GqlRetrieveByFaceID)
	GqlServerType = graphql.NewObject(GqlServerNodeType.Annotate(graphql.ObjectConfig{
		Name:        "FileServer",
		Description: "File server.",
		Fields: tggql.CommonFields(graphql.Fields{
			"counters": &graphql.Field{
				Description: "Counters.",
				Type:        graphql.NewNonNull(GqlCountersType),
				Resolve: func(p graphql.ResolveParams) (interface{}, error) {
					return p.Source.(*Server).Counters(), nil
				},
			},
		}),
	}))
	GqlServerNodeType.Register(GqlServerType)
}
