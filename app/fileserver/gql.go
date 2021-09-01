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
var GqlRetrieveByFaceID func(id iface.ID) interface{}

// GraphQL types.
var (
	GqlMountInput     *graphql.InputObject
	GqlConfigInput    *graphql.InputObject
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

	GqlServerNodeType = tggql.NewNodeType("FileServer", (*Server)(nil), &GqlRetrieveByFaceID)
	GqlServerType = graphql.NewObject(GqlServerNodeType.Annotate(graphql.ObjectConfig{
		Name:        "FileServer",
		Description: "File server.",
		Fields:      tggql.CommonFields(graphql.Fields{}),
	}))
	GqlServerNodeType.Register(GqlServerType)
}
